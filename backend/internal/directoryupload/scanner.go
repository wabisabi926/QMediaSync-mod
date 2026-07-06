package directoryupload

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"qmediasync/internal/helpers"
	"qmediasync/internal/models"
)

// RuleRuntimeMode 是目录监控规则运行模式。
type RuleRuntimeMode string

const (
	RuleRuntimeModeWatcher RuleRuntimeMode = "watcher"
	RuleRuntimeModePolling RuleRuntimeMode = "polling"
)

const (
	defaultProcessedCleanupInterval  = 24 * time.Hour
	defaultProcessedMissingSourceTTL = 30 * 24 * time.Hour
)

// RuleWatcher 是单条目录监控规则的事件监听器。
type RuleWatcher interface {
	Start(ctx context.Context) error
	Close() error
}

// WatcherFactory 创建目录事件监听器。
type WatcherFactory func(service *Service, rule *models.DirectoryUploadRule) (RuleWatcher, error)

// RuleRuntime 保存单条目录监控规则的运行状态。
type RuleRuntime struct {
	RuleID uint
	Mode   RuleRuntimeMode

	cancel  context.CancelFunc
	watcher RuleWatcher
	done    chan struct{}
	once    sync.Once
}

// Stop 停止目录监控规则运行时。
func (runtime *RuleRuntime) Stop() {
	if runtime == nil {
		return
	}
	runtime.once.Do(func() {
		if runtime.cancel != nil {
			runtime.cancel()
		}
		if runtime.watcher != nil {
			_ = runtime.watcher.Close()
		}
		if runtime.done != nil {
			<-runtime.done
		}
	})
}

// Start 启动所有已启用的目录上传规则。
func (service *Service) Start(ctx context.Context) error {
	if service == nil {
		return errors.New("目录上传服务为空")
	}
	rules, err := models.GetEnabledDirectoryUploadRules()
	if err != nil {
		return err
	}
	service.startProcessedCleanup(ctx)
	for _, rule := range rules {
		runtime, err := service.StartRule(ctx, rule)
		if err != nil {
			helpers.AppLogger.Warnf("[目录上传] 启动规则 %d 失败：%v", rule.ID, err)
			continue
		}
		service.mutex.Lock()
		service.runtimes = append(service.runtimes, runtime)
		service.mutex.Unlock()
	}
	return nil
}

// Stop 停止目录上传服务运行的所有规则。
func (service *Service) Stop() {
	if service == nil {
		return
	}
	service.mutex.Lock()
	runtimes := append([]*RuleRuntime(nil), service.runtimes...)
	service.runtimes = nil
	cleanupCancel := service.cleanupCancel
	cleanupDone := service.cleanupDone
	service.cleanupCancel = nil
	service.cleanupDone = nil
	service.mutex.Unlock()
	if cleanupCancel != nil {
		cleanupCancel()
	}
	if cleanupDone != nil {
		<-cleanupDone
	}
	for _, runtime := range runtimes {
		runtime.Stop()
	}
}

// StartRule 启动单条目录上传规则。
func (service *Service) StartRule(ctx context.Context, rule *models.DirectoryUploadRule) (*RuleRuntime, error) {
	if service == nil {
		service = NewService(ServiceOptions{})
	}
	if rule == nil {
		return nil, errors.New("目录监控规则为空")
	}
	ruleCtx, cancel := context.WithCancel(ctx)
	runtime := &RuleRuntime{
		RuleID: rule.ID,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	if rule.StartupScanEnabled {
		if _, err := service.ScanRule(ruleCtx, rule); err != nil {
			cancel()
			return nil, fmt.Errorf("启动补偿扫描失败：%w", err)
		}
	}

	watcher, mode, err := service.startRuleWatcher(ruleCtx, rule)
	if err != nil {
		cancel()
		return nil, err
	}
	runtime.watcher = watcher
	runtime.Mode = mode

	var wg sync.WaitGroup
	if runtime.Mode == RuleRuntimeModePolling {
		wg.Add(1)
		go func() {
			defer wg.Done()
			service.runPollingLoop(ruleCtx, rule)
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		service.runStabilityLoop(ruleCtx, rule)
	}()
	go func() {
		wg.Wait()
		close(runtime.done)
	}()
	return runtime, nil
}

func (service *Service) startRuleWatcher(ctx context.Context, rule *models.DirectoryUploadRule) (RuleWatcher, RuleRuntimeMode, error) {
	switch rule.WatchMode {
	case "", models.DirectoryUploadWatchModeAuto:
	case models.DirectoryUploadWatchModePolling:
		return nil, RuleRuntimeModePolling, nil
	case models.DirectoryUploadWatchModeFSNotify:
	default:
		return nil, "", fmt.Errorf("不支持的监控模式：%s", rule.WatchMode)
	}
	factory := service.watcherFactory
	if factory == nil {
		factory = NewFSNotifyRuleWatcher
	}
	watcher, err := factory(service, rule)
	if err == nil {
		if watcher == nil {
			err = errors.New("watcher 工厂返回空实例")
		} else if err = watcher.Start(ctx); err == nil {
			return watcher, RuleRuntimeModeWatcher, nil
		}
		if watcher != nil {
			_ = watcher.Close()
		}
	}
	if rule.WatchMode == models.DirectoryUploadWatchModeFSNotify {
		return nil, "", fmt.Errorf("启动 watcher 失败：%w", err)
	}
	helpers.AppLogger.Warnf("[目录上传] 规则 %d watcher 不可用，切换为 polling：%v", rule.ID, err)
	return nil, RuleRuntimeModePolling, nil
}

func (service *Service) runPollingLoop(ctx context.Context, rule *models.DirectoryUploadRule) {
	interval := service.pollingInterval(rule)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := service.ScanRule(ctx, rule); err != nil && !errors.Is(err, context.Canceled) {
				helpers.AppLogger.Warnf("[目录上传] 规则 %d polling 扫描失败：%v", rule.ID, err)
			}
		}
	}
}

func (service *Service) runStabilityLoop(ctx context.Context, rule *models.DirectoryUploadRule) {
	interval := service.stabilityInterval(rule)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			service.processStableFiles(ctx, rule)
		}
	}
}

func (service *Service) processStableFiles(ctx context.Context, rule *models.DirectoryUploadRule) {
	files, err := service.CheckStableFiles(rule)
	if err != nil {
		helpers.AppLogger.Warnf("[目录上传] 规则 %d 稳定性检查失败：%v", rule.ID, err)
		return
	}
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := service.HandleStableFile(ctx, rule, file.Path); err != nil {
			helpers.AppLogger.Warnf("[目录上传] 规则 %d 创建上传任务失败：%v", rule.ID, err)
			if ctx.Err() == nil && shouldRequeueStableFile(err) {
				service.TrackPath(rule.ID, file.Path)
			}
		}
	}
}

func shouldRequeueStableFile(err error) bool {
	return err != nil && !errors.Is(err, errStableFileNoRetry) && !errors.Is(err, os.ErrNotExist)
}

func (service *Service) startProcessedCleanup(ctx context.Context) {
	service.cleanupProcessedOnce()

	cleanupCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	service.mutex.Lock()
	if service.cleanupCancel != nil {
		service.mutex.Unlock()
		cancel()
		return
	}
	service.cleanupCancel = cancel
	service.cleanupDone = done
	service.mutex.Unlock()

	go func() {
		defer close(done)
		ticker := time.NewTicker(service.processedCleanupInterval())
		defer ticker.Stop()
		for {
			select {
			case <-cleanupCtx.Done():
				return
			case <-ticker.C:
				service.cleanupProcessedOnce()
			}
		}
	}()
}

func (service *Service) cleanupProcessedOnce() {
	now := service.now()
	service.cleanupExpiredMemoryCaches(now)

	deleted, err := models.CleanupDirectoryUploadProcessedFiles(now, service.processedMissingSourceTTL())
	if err != nil {
		helpers.AppLogger.Warnf("[目录上传] 清理 processed 账本失败：%v", err)
		return
	}
	if deleted > 0 {
		helpers.AppLogger.Infof("[目录上传] 清理 processed 账本记录 %d 条", deleted)
	}
}

func (service *Service) processedCleanupInterval() time.Duration {
	if service.cleanupInterval > 0 {
		return service.cleanupInterval
	}
	return defaultProcessedCleanupInterval
}

func (service *Service) processedMissingSourceTTL() time.Duration {
	if service.missingSourceTTL > 0 {
		return service.missingSourceTTL
	}
	return defaultProcessedMissingSourceTTL
}

func (service *Service) pollingInterval(_ *models.DirectoryUploadRule) time.Duration {
	if service.pollInterval > 0 {
		return service.pollInterval
	}
	return time.Duration(models.DirectoryUploadDefaultRescanIntervalSeconds) * time.Second
}

func (service *Service) stabilityInterval(_ *models.DirectoryUploadRule) time.Duration {
	if service.stabilityCheckInterval > 0 {
		return service.stabilityCheckInterval
	}
	return time.Duration(models.DirectoryUploadDefaultStabilityCheckIntervalSeconds) * time.Second
}

var globalService struct {
	sync.Mutex
	service *Service
	cancel  context.CancelFunc
}

// InitDirectoryUploadService 初始化目录监控上传服务。
func InitDirectoryUploadService() {
	globalService.Lock()
	defer globalService.Unlock()
	if globalService.service != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	service := NewService(ServiceOptions{})
	if err := service.Start(ctx); err != nil {
		helpers.AppLogger.Errorf("[目录上传] 初始化失败：%v", err)
		cancel()
		return
	}
	globalService.service = service
	globalService.cancel = cancel
}

// StopDirectoryUploadService 停止目录监控上传服务。
func StopDirectoryUploadService() {
	globalService.Lock()
	service := globalService.service
	cancel := globalService.cancel
	globalService.service = nil
	globalService.cancel = nil
	globalService.Unlock()
	if cancel != nil {
		cancel()
	}
	if service != nil {
		service.Stop()
	}
}

// ReloadDirectoryUploadService 重载运行中的目录监控上传服务。
func ReloadDirectoryUploadService() {
	globalService.Lock()
	running := globalService.service != nil
	globalService.Unlock()
	if !running {
		return
	}
	StopDirectoryUploadService()
	InitDirectoryUploadService()
}

// ScanRuleNow 使用运行中的目录上传服务执行一次补偿扫描。
func ScanRuleNow(ctx context.Context, rule *models.DirectoryUploadRule) (int, error) {
	globalService.Lock()
	service := globalService.service
	globalService.Unlock()
	if service == nil {
		service = NewService(ServiceOptions{})
	}
	return service.ScanRule(ctx, rule)
}
