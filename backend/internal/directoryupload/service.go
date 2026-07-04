package directoryupload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"qmediasync/internal/helpers"
	"qmediasync/internal/models"
	"qmediasync/internal/v115open"
)

// RemoteDirectory 是目录监控上传目标目录。
type RemoteDirectory struct {
	ID   string
	Path string
}

// RemoteFile 是目录监控上传前检查到的远端文件。
type RemoteFile struct {
	ID       string
	PickCode string
	SHA1     string
	Size     int64
	Mtime    int64
}

// RemoteClient 封装目录监控创建上传任务前需要的远端能力。
type RemoteClient interface {
	EnsureDir(ctx context.Context, rule *models.DirectoryUploadRule, relativeDir string) (RemoteDirectory, error)
	FindFile(ctx context.Context, parentID string, fileName string) (*RemoteFile, error)
	DeleteFile(ctx context.Context, parentID string, fileID string) error
}

// ServiceOptions 是目录监控上传服务依赖。
type ServiceOptions struct {
	Now                    func() time.Time
	RemoteClient           RemoteClient
	WatcherFactory         WatcherFactory
	PollInterval           time.Duration
	StabilityCheckInterval time.Duration
}

type processedFile struct {
	expiresAt time.Time
}

// Service 负责把稳定后的本地文件转换为上传队列任务。
type Service struct {
	now                    func() time.Time
	queue                  *StabilityQueue
	remoteClient           RemoteClient
	watcherFactory         WatcherFactory
	pollInterval           time.Duration
	stabilityCheckInterval time.Duration

	mutex     sync.Mutex
	processed map[string]processedFile
	runtimes  []*RuleRuntime
}

// NewService 创建目录监控上传服务。
func NewService(options ServiceOptions) *Service {
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		now:                    now,
		queue:                  NewStabilityQueue(StabilityQueueOptions{Now: now}),
		remoteClient:           options.RemoteClient,
		watcherFactory:         options.WatcherFactory,
		pollInterval:           options.PollInterval,
		stabilityCheckInterval: options.StabilityCheckInterval,
		processed:              make(map[string]processedFile),
	}
}

// SetRemoteClient 设置远端客户端，主要供测试替换。
func (service *Service) SetRemoteClient(client RemoteClient) {
	if service == nil {
		return
	}
	service.mutex.Lock()
	defer service.mutex.Unlock()
	service.remoteClient = client
}

// PendingPaths 返回指定规则当前待稳定文件路径。
func (service *Service) PendingPaths(ruleID uint) []string {
	if service == nil || service.queue == nil {
		return []string{}
	}
	return service.queue.PendingPaths(ruleID)
}

// TrackPath 将文件加入稳定性检查队列。
func (service *Service) TrackPath(ruleID uint, path string) {
	if service == nil || service.queue == nil {
		return
	}
	service.queue.Track(ruleID, path)
}

func (service *Service) trackCandidatePath(ctx context.Context, rule *models.DirectoryUploadRule, path string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if rule == nil {
		return false, errors.New("目录监控规则为空")
	}
	syncPath := models.GetSyncPathById(rule.SyncPathId)
	if syncPath == nil {
		return false, fmt.Errorf("同步目录不存在：%d", rule.SyncPathId)
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.IsDir() {
		return false, nil
	}
	rel, err := safeRelativePath(rule.MonitorPath, path)
	if err != nil {
		return false, err
	}
	if shouldIgnorePath(rel, info.Name(), false, parseIgnorePatterns(rule.IgnorePatternsStr)) {
		return false, nil
	}
	if !syncPath.IsValidVideoExt(info.Name()) {
		return false, nil
	}
	service.queue.Track(rule.ID, path)
	return true, nil
}

// CheckStableFiles 执行指定规则的一轮稳定性检查。
func (service *Service) CheckStableFiles(rule *models.DirectoryUploadRule) ([]StableFile, error) {
	if service == nil || service.queue == nil {
		return []StableFile{}, nil
	}
	return service.queue.Check(rule)
}

// ScanRule 扫描目录上传规则，把候选视频文件加入稳定性队列。
func (service *Service) ScanRule(ctx context.Context, rule *models.DirectoryUploadRule) (int, error) {
	if service == nil {
		service = NewService(ServiceOptions{})
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if rule == nil {
		return 0, errors.New("目录监控规则为空")
	}
	syncPath := models.GetSyncPathById(rule.SyncPathId)
	if syncPath == nil {
		return 0, fmt.Errorf("同步目录不存在：%d", rule.SyncPathId)
	}
	monitorPath := filepath.Clean(rule.MonitorPath)
	info, err := os.Stat(monitorPath)
	if err != nil {
		return 0, fmt.Errorf("读取监控目录失败：%w", err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("监控路径不是目录：%s", monitorPath)
	}

	accepted := 0
	ignorePatterns := parseIgnorePatterns(rule.IgnorePatternsStr)
	walkErr := filepath.WalkDir(monitorPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == monitorPath {
			return nil
		}
		rel, err := filepath.Rel(monitorPath, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if !rule.Recursive {
				return filepath.SkipDir
			}
			if shouldIgnorePath(rel, entry.Name(), true, ignorePatterns) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldIgnorePath(rel, entry.Name(), false, ignorePatterns) || !syncPath.IsValidVideoExt(entry.Name()) {
			return nil
		}
		service.queue.Track(rule.ID, path)
		accepted++
		return nil
	})
	if walkErr != nil {
		return accepted, walkErr
	}
	return accepted, nil
}

// HandleStableFile 为已稳定的本地文件创建上传任务。
func (service *Service) HandleStableFile(ctx context.Context, rule *models.DirectoryUploadRule, filePath string) error {
	if service == nil {
		service = NewService(ServiceOptions{})
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if rule == nil {
		return errors.New("目录监控规则为空")
	}
	filePath = filepath.Clean(filePath)
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("读取稳定文件失败：%w", err)
	}
	if info.IsDir() {
		return nil
	}
	syncPath := models.GetSyncPathById(rule.SyncPathId)
	if syncPath == nil {
		return fmt.Errorf("同步目录不存在：%d", rule.SyncPathId)
	}
	if !syncPath.IsValidVideoExt(info.Name()) {
		return nil
	}
	rel, err := safeRelativePath(rule.MonitorPath, filePath)
	if err != nil {
		return err
	}
	signature := fileSignature(info)
	if service.isProcessed(rule, rel, signature) {
		return nil
	}

	relativeDir := filepath.ToSlash(filepath.Dir(rel))
	if relativeDir == "." {
		relativeDir = ""
	}
	remoteClient, err := service.remote(ctx, rule)
	if err != nil {
		return err
	}
	remoteDir, err := remoteClient.EnsureDir(ctx, rule, relativeDir)
	if err != nil {
		return fmt.Errorf("确认远端目录失败：%w", err)
	}
	fileName := info.Name()
	remoteFilePath := cleanRemoteFilePath(rule.RemoteRootPath, rel)
	task := &models.DbUploadTask{
		Source:              models.UploadSourceDirectoryMonitor,
		AccountId:           rule.AccountId,
		SyncPathId:          rule.SyncPathId,
		SourceType:          models.SourceType115,
		LocalFullPath:       filePath,
		RelativePath:        filepath.ToSlash(rel),
		RemoteFileId:        remoteFilePath,
		RemotePathId:        remoteDir.ID,
		FileName:            fileName,
		Status:              models.UploadStatusPending,
		FileSize:            info.Size(),
		UploadResult:        models.UploadResultUnknown,
		SourceCleanupStatus: cleanupInitialStatus(rule),
	}

	remoteFile, err := remoteClient.FindFile(ctx, remoteDir.ID, fileName)
	if err != nil {
		return fmt.Errorf("检查远端同名文件失败：%w", err)
	}
	if remoteFile != nil && remoteFile.ID != "" {
		if isSameRemoteFile(remoteFile, filePath, info.Size()) {
			task.Status = models.UploadStatusCompleted
			task.UploadResult = models.UploadResultRemoteExists
			task.UploadedBytes = info.Size()
			task.CompletedRemoteFileId = remoteFile.ID
			task.CompletedPickCode = remoteFile.PickCode
			task.EndTime = service.now().Unix()
			if err := models.AddDirectoryMonitorUploadTask(task); err != nil {
				return fmt.Errorf("创建远端已存在上传任务失败：%w", err)
			}
			if err := task.EnqueueStrmGenerationAfterUpload(); err != nil {
				return fmt.Errorf("创建 STRM 生成任务失败：%w", err)
			}
			service.markProcessed(rule, rel, signature)
			return nil
		}

		switch rule.OverwriteMode {
		case "", models.DirectoryUploadOverwriteSkipSame:
			service.markProcessed(rule, rel, signature)
			return nil
		case models.DirectoryUploadOverwriteFailConflict:
			return fmt.Errorf("远端已存在同名文件且大小或 SHA1 不一致：%s", remoteFilePath)
		case models.DirectoryUploadOverwriteReplaceConflict:
			if err := remoteClient.DeleteFile(ctx, remoteDir.ID, remoteFile.ID); err != nil {
				return fmt.Errorf("删除远端同名文件失败：%w", err)
			}
		default:
			return fmt.Errorf("不支持的同名文件处理方式：%s", rule.OverwriteMode)
		}
	}

	if err := models.AddDirectoryMonitorUploadTask(task); err != nil {
		return fmt.Errorf("创建目录监控上传任务失败：%w", err)
	}
	service.markProcessed(rule, rel, signature)
	return nil
}

func (service *Service) remote(ctx context.Context, rule *models.DirectoryUploadRule) (RemoteClient, error) {
	service.mutex.Lock()
	client := service.remoteClient
	service.mutex.Unlock()
	if client != nil {
		return client, nil
	}
	return newOpen115RemoteClient(ctx, rule)
}

func (service *Service) isProcessed(rule *models.DirectoryUploadRule, rel string, signature string) bool {
	key := processedKey(rule.ID, rel, signature)
	now := service.now()
	service.mutex.Lock()
	defer service.mutex.Unlock()
	item, ok := service.processed[key]
	if !ok {
		return false
	}
	if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
		delete(service.processed, key)
		return false
	}
	return true
}

func (service *Service) markProcessed(rule *models.DirectoryUploadRule, rel string, signature string) {
	key := processedKey(rule.ID, rel, signature)
	ttl := time.Duration(rule.ProcessedCacheTTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	service.mutex.Lock()
	defer service.mutex.Unlock()
	service.processed[key] = processedFile{expiresAt: service.now().Add(ttl)}
}

func processedKey(ruleID uint, rel string, signature string) string {
	return fmt.Sprintf("%d:%s:%s", ruleID, filepath.ToSlash(rel), signature)
}

func cleanupInitialStatus(rule *models.DirectoryUploadRule) models.UploadSourceCleanupStatus {
	if rule != nil && rule.DeleteSourceAfterSuccess {
		return models.UploadSourceCleanupStatusPending
	}
	return models.UploadSourceCleanupStatusNone
}

func cleanRemoteFilePath(rootPath string, rel string) string {
	rootPath = strings.ReplaceAll(strings.TrimSpace(rootPath), "\\", "/")
	if rootPath == "" {
		rootPath = "/"
	}
	if !strings.HasPrefix(rootPath, "/") {
		rootPath = "/" + rootPath
	}
	parts := []string{rootPath}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	return pathpkg.Clean(pathpkg.Join(parts...))
}

func safeRelativePath(basePath string, path string) (string, error) {
	basePath = filepath.Clean(basePath)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(basePath, path)
	if err != nil {
		return "", fmt.Errorf("计算相对路径失败：%w", err)
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", fmt.Errorf("文件路径越界：%s", path)
	}
	return filepath.ToSlash(rel), nil
}

func parseIgnorePatterns(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var patterns []string
	if err := json.Unmarshal([]byte(raw), &patterns); err == nil {
		return compactPatterns(patterns)
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';'
	})
	return compactPatterns(fields)
}

func compactPatterns(patterns []string) []string {
	result := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			result = append(result, pattern)
		}
	}
	return result
}

func shouldIgnorePath(rel string, name string, isDir bool, patterns []string) bool {
	if name == "" {
		return false
	}
	if strings.HasPrefix(name, ".") {
		return true
	}
	lowerName := strings.ToLower(name)
	for _, suffix := range []string{".part", ".tmp", ".download"} {
		if strings.HasSuffix(lowerName, suffix) {
			return true
		}
	}
	rel = filepath.ToSlash(rel)
	for _, pattern := range patterns {
		if matched, _ := pathpkg.Match(pattern, rel); matched {
			return true
		}
		if matched, _ := pathpkg.Match(pattern, name); matched {
			return true
		}
		if isDir && strings.TrimSuffix(pattern, "/") == rel {
			return true
		}
	}
	return false
}

func isSameRemoteFile(remoteFile *RemoteFile, localPath string, size int64) bool {
	if remoteFile == nil || remoteFile.ID == "" {
		return false
	}
	if remoteFile.Size != size {
		return false
	}
	if strings.TrimSpace(remoteFile.SHA1) == "" {
		return false
	}
	localSHA1, err := helpers.FileSHA1(localPath)
	if err != nil {
		return false
	}
	return strings.EqualFold(remoteFile.SHA1, localSHA1)
}

type open115RemoteClient struct {
	client *v115open.OpenClient
}

func newOpen115RemoteClient(_ context.Context, rule *models.DirectoryUploadRule) (RemoteClient, error) {
	if rule == nil {
		return nil, errors.New("目录监控规则为空")
	}
	account, err := models.GetAccountById(rule.AccountId)
	if err != nil {
		return nil, fmt.Errorf("获取 115 账号失败：%w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("账号不存在：%d", rule.AccountId)
	}
	client := account.Get115Client()
	if client == nil {
		return nil, fmt.Errorf("账号 %d 未初始化 115 客户端", rule.AccountId)
	}
	return &open115RemoteClient{client: client}, nil
}

func (client *open115RemoteClient) EnsureDir(ctx context.Context, rule *models.DirectoryUploadRule, relativeDir string) (RemoteDirectory, error) {
	if client == nil || client.client == nil {
		return RemoteDirectory{}, errors.New("115 远端客户端为空")
	}
	if rule == nil {
		return RemoteDirectory{}, errors.New("目录监控规则为空")
	}
	parentID := rule.RemoteRootId
	if parentID == "" {
		return RemoteDirectory{}, errors.New("远端根目录 ID 为空")
	}
	rootPath := strings.TrimSpace(rule.RemoteRootPath)
	if rootPath == "" {
		rootPath = "/"
	}
	relativeDir = strings.Trim(filepath.ToSlash(relativeDir), "/")
	if relativeDir == "" || relativeDir == "." {
		return RemoteDirectory{ID: parentID, Path: cleanRemoteFilePath(rootPath, "")}, nil
	}

	currentPath := strings.TrimSuffix(cleanRemoteFilePath(rootPath, ""), "/")
	if currentPath == "." {
		currentPath = "/"
	}
	for _, segment := range strings.Split(relativeDir, "/") {
		segment = strings.TrimSpace(segment)
		if segment == "" || segment == "." {
			continue
		}
		nextPath := cleanRemoteFilePath(currentPath, segment)
		detail, err := client.client.GetFsDetailByPath(ctx, nextPath)
		if err == nil && detail != nil && detail.FileId != "" {
			parentID = detail.FileId
			currentPath = nextPath
			continue
		}
		createdID, err := client.client.MkDir(ctx, parentID, segment)
		if err != nil {
			return RemoteDirectory{}, fmt.Errorf("创建远端目录 %s 失败：%w", nextPath, err)
		}
		parentID = createdID
		currentPath = nextPath
	}
	return RemoteDirectory{ID: parentID, Path: currentPath}, nil
}

func (client *open115RemoteClient) FindFile(ctx context.Context, parentID string, fileName string) (*RemoteFile, error) {
	if client == nil || client.client == nil {
		return nil, errors.New("115 远端客户端为空")
	}
	if parentID == "" || fileName == "" {
		return nil, nil
	}
	const pageSize = 1150
	for offset := 0; ; offset += pageSize {
		resp, err := client.client.GetFsList(ctx, parentID, true, false, false, offset, pageSize)
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.Data) == 0 {
			return nil, nil
		}
		for _, item := range resp.Data {
			if item.FileCategory != v115open.TypeFile || item.FileName != fileName {
				continue
			}
			mtime := item.Ptime
			if mtime == 0 {
				mtime = item.Utime
			}
			return &RemoteFile{
				ID:       item.FileId,
				PickCode: item.PickCode,
				SHA1:     item.Sha1,
				Size:     item.FileSize,
				Mtime:    mtime,
			}, nil
		}
		if len(resp.Data) < pageSize {
			return nil, nil
		}
	}
}

func (client *open115RemoteClient) DeleteFile(ctx context.Context, parentID string, fileID string) error {
	if client == nil || client.client == nil {
		return errors.New("115 远端客户端为空")
	}
	if parentID == "" || fileID == "" {
		return errors.New("远端文件信息为空")
	}
	success, err := client.client.Del(ctx, []string{fileID}, parentID)
	if err != nil {
		return err
	}
	if !success {
		return errors.New("115 删除接口返回失败")
	}
	return nil
}
