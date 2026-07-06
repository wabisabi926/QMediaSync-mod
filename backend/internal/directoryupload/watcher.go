package directoryupload

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"qmediasync/internal/helpers"
	"qmediasync/internal/models"

	"github.com/fsnotify/fsnotify"
)

type fsNotifyRuleWatcher struct {
	service *Service
	rule    *models.DirectoryUploadRule

	mutex   sync.Mutex
	watcher *fsnotify.Watcher
	once    sync.Once
}

// NewFSNotifyRuleWatcher 创建基于 fsnotify 的目录监控 watcher。
func NewFSNotifyRuleWatcher(service *Service, rule *models.DirectoryUploadRule) (RuleWatcher, error) {
	if service == nil {
		return nil, errors.New("目录上传服务为空")
	}
	if rule == nil {
		return nil, errors.New("目录监控规则为空")
	}
	return &fsNotifyRuleWatcher{service: service, rule: rule}, nil
}

func (watcher *fsNotifyRuleWatcher) Start(ctx context.Context) error {
	if watcher == nil {
		return errors.New("watcher 为空")
	}
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	watcher.mutex.Lock()
	watcher.watcher = fsWatcher
	watcher.mutex.Unlock()
	if err := watcher.addRecursive(watcher.rule.MonitorPath); err != nil {
		_ = fsWatcher.Close()
		return err
	}
	go watcher.run(ctx)
	return nil
}

func (watcher *fsNotifyRuleWatcher) Close() error {
	if watcher == nil {
		return nil
	}
	var err error
	watcher.once.Do(func() {
		watcher.mutex.Lock()
		fsWatcher := watcher.watcher
		watcher.watcher = nil
		watcher.mutex.Unlock()
		if fsWatcher != nil {
			err = fsWatcher.Close()
		}
	})
	return err
}

func (watcher *fsNotifyRuleWatcher) run(ctx context.Context) {
	for {
		watcher.mutex.Lock()
		fsWatcher := watcher.watcher
		watcher.mutex.Unlock()
		if fsWatcher == nil {
			return
		}
		select {
		case <-ctx.Done():
			_ = watcher.Close()
			return
		case event, ok := <-fsWatcher.Events:
			if !ok {
				return
			}
			watcher.handleEvent(ctx, event)
		case err, ok := <-fsWatcher.Errors:
			if !ok {
				return
			}
			helpers.AppLogger.Warnf("[目录上传] watcher 错误：%v", err)
		}
	}
}

func (watcher *fsNotifyRuleWatcher) handleEvent(ctx context.Context, event fsnotify.Event) {
	if event.Name == "" {
		return
	}
	if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
		return
	}
	if event.Op&fsnotify.Create == fsnotify.Create {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			if watcher.rule == nil || !watcher.rule.Recursive {
				return
			}
			ignorePatterns := parseIgnorePatterns(watcher.rule.IgnorePatternsStr)
			shouldWatch, err := watcher.shouldWatchDirectory(event.Name, ignorePatterns)
			if err != nil {
				helpers.AppLogger.Warnf("[目录上传] 新目录路径越界：%v", err)
				return
			}
			if !shouldWatch {
				return
			}
			if err := watcher.addRecursive(event.Name); err != nil {
				helpers.AppLogger.Warnf("[目录上传] 新目录加入 watcher 失败：%v", err)
			}
			go func(path string) {
				if _, err := watcher.service.ScanSubtree(ctx, watcher.rule, path); err != nil && !errors.Is(err, context.Canceled) {
					helpers.AppLogger.Warnf("[目录上传] 新目录补偿扫描失败：%v", err)
				}
			}(event.Name)
			return
		}
	}
	if _, err := watcher.service.trackCandidatePath(ctx, watcher.rule, event.Name); err != nil && !errors.Is(err, context.Canceled) {
		helpers.AppLogger.Warnf("[目录上传] watcher 处理文件事件失败：%v", err)
	}
}

func (watcher *fsNotifyRuleWatcher) addRecursive(root string) error {
	root = filepath.Clean(root)
	ignorePatterns := parseIgnorePatterns(watcher.rule.IgnorePatternsStr)
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		shouldWatch, err := watcher.shouldWatchDirectory(path, ignorePatterns)
		if err != nil {
			return err
		}
		if !shouldWatch {
			return filepath.SkipDir
		}
		watcher.mutex.Lock()
		fsWatcher := watcher.watcher
		watcher.mutex.Unlock()
		if fsWatcher == nil {
			return errors.New("watcher 已关闭")
		}
		return fsWatcher.Add(path)
	})
}

func (watcher *fsNotifyRuleWatcher) shouldWatchDirectory(path string, ignorePatterns []string) (bool, error) {
	if watcher == nil || watcher.rule == nil {
		return false, errors.New("目录监控规则为空")
	}
	rel, err := relativePathInMonitor(watcher.rule.MonitorPath, path)
	if err != nil {
		return false, err
	}
	if rel == "." {
		return true, nil
	}
	if shouldIgnorePath(rel, filepath.Base(path), true, ignorePatterns) {
		return false, nil
	}
	if !watcher.rule.Recursive {
		return false, nil
	}
	return true, nil
}
