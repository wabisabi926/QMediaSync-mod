package helpers

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type AdvancedFolderWatcher struct {
	watcher   *fsnotify.Watcher
	watchPath string
	// 文件过滤
	extensions []string // 监控的文件扩展名
	ignoreDirs []string // 忽略的目录
	// 事件去重
	eventCache map[string]time.Time
	debounce   time.Duration
}

func NewAdvancedFolderWatcher(path string) *AdvancedFolderWatcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	return &AdvancedFolderWatcher{
		watcher:    watcher,
		watchPath:  path,
		extensions: []string{".txt", ".go", ".json", ".yaml", ".yml"},
		ignoreDirs: []string{".git", ".vscode", "node_modules", "__pycache__"},
		eventCache: make(map[string]time.Time),
		debounce:   100 * time.Millisecond,
	}
}

// shouldIgnore 检查是否应该忽略该路径
func (afw *AdvancedFolderWatcher) shouldIgnore(path string) bool {
	// 检查是否在忽略目录中
	for _, ignoreDir := range afw.ignoreDirs {
		if strings.Contains(path, ignoreDir) {
			return true
		}
	}

	// 检查文件扩展名
	if !afw.isWatchedExtension(path) {
		return true
	}

	return false
}

// isWatchedExtension 检查是否是监控的文件扩展名
func (afw *AdvancedFolderWatcher) isWatchedExtension(path string) bool {
	if len(afw.extensions) == 0 {
		return true // 监控所有文件
	}

	ext := strings.ToLower(filepath.Ext(path))
	for _, watchedExt := range afw.extensions {
		if ext == watchedExt {
			return true
		}
	}
	return false
}

// addWatchRecursive 递归添加监控
func (afw *AdvancedFolderWatcher) addWatchRecursive(path string) error {
	return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// 跳过忽略的目录
			if afw.shouldIgnore(walkPath) {
				return filepath.SkipDir
			}

			err = afw.watcher.Add(walkPath)
			if err != nil {
				log.Printf("警告: 无法监控目录 %s: %v", walkPath, err)
			}
		}
		return nil
	})
}

// processEvent 处理事件（带去重）
func (afw *AdvancedFolderWatcher) processEvent(event fsnotify.Event) {
	// 检查是否应该忽略
	if afw.shouldIgnore(event.Name) {
		return
	}

	// 事件去重
	now := time.Now()
	if lastTime, exists := afw.eventCache[event.Name]; exists {
		if now.Sub(lastTime) < afw.debounce {
			return // 忽略短时间内重复的事件
		}
	}
	afw.eventCache[event.Name] = now

	// 清理过期的缓存
	for path, lastTime := range afw.eventCache {
		if now.Sub(lastTime) > time.Minute {
			delete(afw.eventCache, path)
		}
	}

	// 处理事件
	afw.handleEvent(event)
}

// handleEvent 处理具体事件
func (afw *AdvancedFolderWatcher) handleEvent(event fsnotify.Event) {
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		afw.handleCreate(event)
	case event.Op&fsnotify.Write == fsnotify.Write:
		afw.handleWrite(event)
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		afw.handleRemove(event)
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		afw.handleRename(event)
	}
}

func (afw *AdvancedFolderWatcher) handleCreate(event fsnotify.Event) {
	info, err := os.Stat(event.Name)
	if err != nil {
		return
	}

	if info.IsDir() {
		log.Printf("📁 新目录创建: %s", event.Name)
		afw.addWatchRecursive(event.Name)
	} else {
		log.Printf("📄 新文件创建: %s", event.Name)
	}
}

func (afw *AdvancedFolderWatcher) handleWrite(event fsnotify.Event) {
	info, err := os.Stat(event.Name)
	if err != nil || info.IsDir() {
		return
	}
	log.Printf("✏️  文件修改: %s (大小: %d bytes)", event.Name, info.Size())
}

func (afw *AdvancedFolderWatcher) handleRemove(event fsnotify.Event) {
	log.Printf("🗑️  文件/目录删除: %s", event.Name)
}

func (afw *AdvancedFolderWatcher) handleRename(event fsnotify.Event) {
	log.Printf("📝 文件重命名: %s", event.Name)
}

// Start 开始监控
func (afw *AdvancedFolderWatcher) Start() {
	// 初始添加监控
	if err := afw.addWatchRecursive(afw.watchPath); err != nil {
		log.Fatal(err)
	}

	log.Printf("🚀 开始高级监控目录: %s", afw.watchPath)
	log.Printf("📊 监控文件类型: %v", afw.extensions)
	log.Printf("🚫 忽略目录: %v", afw.ignoreDirs)

	for {
		select {
		case event, ok := <-afw.watcher.Events:
			if !ok {
				return
			}
			afw.processEvent(event)

		case err, ok := <-afw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("❌ 监控错误: %v", err)
		}
	}
}

// Close 关闭
func (afw *AdvancedFolderWatcher) Close() {
	afw.watcher.Close()
}

// func main() {
//     watchPath := "."
//     if len(os.Args) > 1 {
//         watchPath = os.Args[1]
//     }

//     watcher := NewAdvancedFolderWatcher(watchPath)
//     defer watcher.Close()

//     watcher.Start()
// }
