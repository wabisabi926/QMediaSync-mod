package models

import (
	"Q115-STRM/internal/helpers"
	"sync"
	"time"
)

// UploadQueue 上传队列
type UQ struct {
	tasks      chan *DbUploadTask // 所有待上传任务
	numWorkers int                // 工作线程数
	mutex      sync.RWMutex       // 读写锁，保护TaskMap
	running    bool               // 队列是否正在运行
}

// 全局上传队列实例
var GlobalUploadQueue *UQ

// InitUploadQueueBySync 初始化上传队列
func InitUQ() bool {
	if GlobalUploadQueue != nil {
		// 有正在运行的队列
		helpers.AppLogger.Warnf("上传队列已存在，无法重复初始化")
		return false
	}
	GlobalUploadQueue = NewUq(1)
	GlobalUploadQueue.Start()
	helpers.AppLogger.Infof("上传队列初始化完成，工作线程数 %d", GlobalUploadQueue.numWorkers)
	return true
}

func NewUq(maxConcurrency int) *UQ {
	return &UQ{
		tasks:      make(chan *DbUploadTask, maxConcurrency),
		numWorkers: maxConcurrency,
		running:    false,
	}
}

// Start 启动上传队列
func (uq *UQ) Start() {
	uq.mutex.Lock()
	if uq.running {
		uq.mutex.Unlock()
		return
	}
	uq.running = true
	// 重新创建tasks通道和results通道
	uq.tasks = make(chan *DbUploadTask, uq.numWorkers)
	uq.mutex.Unlock()
	// 启动工作协程
	for i := 0; i < uq.numWorkers; i++ {
		go uq.worker()
	}

	// 启动任务调度协程
	go uq.taskScheduler()
}

// worker 执行上传任务
func (uq *UQ) worker() {
	for {
		// 检查队列是否仍在运行
		uq.mutex.RLock()
		running := uq.running
		uq.mutex.RUnlock()

		if !running {
			// helpers.AppLogger.Debug("上传队列已停止，工作协程退出")
			return
		}

		// 尝试从任务通道获取任务
		select {
		case task, ok := <-uq.tasks:
			if !ok {
				// 通道已关闭，退出工作协程
				// helpers.AppLogger.Debug("任务通道已关闭，工作协程退出")
				return
			}
			// 在新的goroutine中处理任务
			task.Upload()
		case <-time.After(100 * time.Millisecond):
			// 超时检查，避免无限阻塞
		}
	}
}

// 任务调度器，定期将TaskMap中的任务加入到tasks通道
// 将TaskMap中的任务移动到tasks通道
func (uq *UQ) moveTasksToChannel() {
	// 检查队列是否正在运行
	uq.mutex.RLock()
	running := uq.running
	uq.mutex.RUnlock()

	if !running {
		return
	}

	// 检查通道是否已满
	uq.mutex.RLock()
	if len(uq.tasks) >= cap(uq.tasks) {
		uq.mutex.RUnlock()
		return
	}
	uq.mutex.RUnlock()

	// 获取一些任务加入到通道
	uq.mutex.Lock()
	defer uq.mutex.Unlock()

	// 计算需要移动的任务数量
	availableSpace := cap(uq.tasks) - len(uq.tasks)
	tasksToMove := availableSpace
	// 从数据库中查询tasksToMove条待上传的记录
	tasks := GetPendingUploadTasks(tasksToMove)
	if len(tasks) == 0 {
		return
	}
	// helpers.AppLogger.Infof("任务调度器：队列容量 %d，当前队列长度 %d，可用空间 %d，待移动任务数 %d", cap(uq.tasks), len(uq.tasks), availableSpace, tasksToMove)
	// 移动任务到通道
	movedCount := 0
outer:
	for _, task := range tasks {
		// 尝试将任务发送到通道
		select {
		case uq.tasks <- task:
			movedCount++
		default:
			// 通道已满，停止移动任务
			break outer
		}
	}

	// if movedCount > 0 {
	// 	helpers.AppLogger.Debugf("已将 %d 个任务添加到上传通道", movedCount)
	// }
}

// taskScheduler 定时将TaskMap中的任务移动到tasks通道
func (uq *UQ) taskScheduler() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		// 检查队列是否仍在运行
		uq.mutex.RLock()
		running := uq.running
		uq.mutex.RUnlock()

		if !running {
			// helpers.AppLogger.Debug("上传队列已停止，任务调度器退出")
			return
		}

		<-ticker.C
		uq.moveTasksToChannel()
	}
}

// Stop 停止上传队列
func (uq *UQ) Stop() {
	uq.mutex.Lock()
	if !uq.running {
		uq.mutex.Unlock()
		helpers.AppLogger.Warnf("上传队列未在运行中")
		return
	}
	uq.running = false
	uq.mutex.Unlock()

	// 关闭tasks通道
	close(uq.tasks)

	helpers.AppLogger.Info("上传队列已停止")
}

// Restart 重启上传队列
func (uq *UQ) Restart() {
	uq.Stop()
	uq.Start()
	helpers.AppLogger.Info("上传队列已重启")
}

func (uq *UQ) IsRunning() bool {
	uq.mutex.RLock()
	defer uq.mutex.RUnlock()
	return uq.running
}
