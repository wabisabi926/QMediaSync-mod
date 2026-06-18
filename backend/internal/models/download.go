package models

import (
	"Q115-STRM/internal/db"
	"Q115-STRM/internal/helpers"
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// DownloadQueue 下载队列
type DQ struct {
	tasks      chan *DbDownloadTask // 所有待下载任务
	numWorkers int                  // 工作线程数
	mutex      sync.RWMutex         // 读写锁，保护TaskMap
	running    bool                 // 队列是否正在运行
	limiter    *rate.Limiter        // 限速器，控制QPS
}

// String 返回状态的字符串表示
func (s DownloadStatus) String() string {
	switch s {
	case DownloadStatusPending:
		return "待下载"
	case DownloadStatusDownloading:
		return "下载中"
	case DownloadStatusCompleted:
		return "已完成"
	case DownloadStatusFailed:
		return "失败"
	case DownloadStatusCancelled:
		return "已取消"
	default:
		return "未知"
	}
}

// 全局下载队列实例
var GlobalDownloadQueue *DQ

// InitDownloadQueueBySync 初始化下载队列
func InitDQ() bool {
	if GlobalDownloadQueue != nil && len(GlobalDownloadQueue.tasks) > 0 {
		// 停止队列
		GlobalDownloadQueue.Stop()
	}
	GlobalDownloadQueue = NewDq(SettingsGlobal.DownloadThreads)
	GlobalDownloadQueue.Start()
	helpers.AppLogger.Infof("下载队列初始化完成，工作线程数 %d", GlobalDownloadQueue.numWorkers)
	return true
}

func NewDq(maxConcurrency int) *DQ {
	return &DQ{
		tasks:      make(chan *DbDownloadTask, maxConcurrency+2),
		numWorkers: maxConcurrency,
		running:    false,
		limiter:    rate.NewLimiter(rate.Limit(maxConcurrency), maxConcurrency),
	}
}

// 启动下载队列的工作协程
func (dq *DQ) Start() {
	// 重新创建tasks通道
	dq.mutex.Lock()
	dq.tasks = make(chan *DbDownloadTask, dq.numWorkers*10)
	dq.mutex.Unlock()
	// 将所有的下载中改为待下载
	db.Db.Model(&DbDownloadTask{}).Where("status = ?", DownloadStatusDownloading).Update("status", DownloadStatusPending)
	dq.mutex.Lock()
	if dq.running {
		dq.mutex.Unlock()
		helpers.AppLogger.Warnf("下载队列已在运行中")
		return
	}
	dq.running = true
	dq.mutex.Unlock()

	// 启动工作协程
	for i := 0; i < dq.numWorkers; i++ {
		go dq.worker()
	}

	// 启动任务调度协程
	go dq.taskScheduler()
}

// Stop 停止下载队列
func (dq *DQ) Stop() {
	dq.mutex.Lock()
	if !dq.running {
		dq.mutex.Unlock()
		helpers.AppLogger.Warnf("下载队列未在运行中")
		return
	}
	dq.running = false
	dq.mutex.Unlock()

	// 关闭tasks通道
	close(dq.tasks)

	helpers.AppLogger.Info("下载队列已停止")
}

// Restart 重启下载队列
func (dq *DQ) Restart() {
	dq.Stop()
	dq.Start()
	helpers.AppLogger.Info("下载队列已重启")
}

// UpdateConcurrency 更新并发数
func (dq *DQ) UpdateConcurrency(newConcurrency int) {
	if newConcurrency <= 0 {
		helpers.AppLogger.Errorf("无效的并发数: %d", newConcurrency)
		return
	}

	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	oldConcurrency := dq.numWorkers
	dq.numWorkers = newConcurrency

	// 更新限速器
	if dq.limiter != nil {
		dq.limiter.SetLimit(rate.Limit(newConcurrency))
		dq.limiter.SetBurst(newConcurrency)
	}

	// 如果并发数增加了，需要启动新的 worker
	if newConcurrency > oldConcurrency {
		for i := oldConcurrency; i < newConcurrency; i++ {
			go dq.worker()
		}
		helpers.AppLogger.Infof("下载队列并发数从 %d 增加到 %d", oldConcurrency, newConcurrency)
	} else if newConcurrency < oldConcurrency {
		// 如果并发数减少了，多余的工作协程会在下次循环中自动退出
		helpers.AppLogger.Infof("下载队列并发数从 %d 减少到 %d", oldConcurrency, newConcurrency)
	}
}

// GetConcurrency 获取当前并发数
func (dq *DQ) GetConcurrency() int {
	dq.mutex.RLock()
	defer dq.mutex.RUnlock()
	return dq.numWorkers
}

// 任务调度器，定期将TaskMap中的任务加入到tasks通道
// taskScheduler 定时将TaskMap中的任务移动到tasks通道
func (dq *DQ) taskScheduler() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		// 检查队列是否仍在运行
		dq.mutex.RLock()
		running := dq.running
		dq.mutex.RUnlock()

		if !running {
			return
		}

		<-ticker.C
		dq.moveTasksToChannel()
	}
}

// 将TaskMap中的任务移动到tasks通道
func (dq *DQ) moveTasksToChannel() {
	// 检查队列是否正在运行
	dq.mutex.RLock()
	running := dq.running
	dq.mutex.RUnlock()

	if !running {
		return
	}

	// 检查通道是否已满
	dq.mutex.RLock()
	if len(dq.tasks) >= cap(dq.tasks) {
		dq.mutex.RUnlock()
		return
	}
	dq.mutex.RUnlock()

	// 获取一些任务加入到通道
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	// 查询待下载的总量
	var total int64
	db.Db.Model(&DbDownloadTask{}).Where("status = ?", DownloadStatusPending).Count(&total)
	if total == 0 {
		return
	}
	// 计算需要移动的任务数量
	availableSpace := cap(dq.tasks) - len(dq.tasks)
	tasksToMove := min(availableSpace, int(total))
	// helpers.AppLogger.Infof("任务调度器：队列容量 %d，当前队列长度 %d，可用空间 %d，待移动任务数 %d", cap(dq.tasks), len(dq.tasks), availableSpace, tasksToMove)
	// 移动任务到通道
	// 从数据库中查询tasksToMove条待下载的记录
	tasks := GetPendingDownloadTasks(tasksToMove)
	if len(tasks) == 0 {
		return
	}
	movedCount := 0
outer:
	for _, task := range tasks {
		// 尝试将任务发送到通道
		select {
		case dq.tasks <- task:
			movedCount++
			// 将任务标记为下载中， 防止重复添加
			task.Downloading()
		default:
			// 通道已满，停止移动任务
			break outer
		}
	}
	// if movedCount > 0 {
	// 	helpers.AppLogger.Debugf("已将 %d 个任务添加到下载通道", movedCount)
	// }
}

// 执行下载任务
// 工作协程
func (dq *DQ) worker() {
	for {
		// 检查队列是否仍在运行
		dq.mutex.RLock()
		running := dq.running
		dq.mutex.RUnlock()

		if !running {
			break
		}

		// 等待限速器令牌（所有worker共享同一个limiter）
		if err := dq.limiter.Wait(context.Background()); err != nil {
			helpers.AppLogger.Errorf("等待限速器失败: %v", err)
			continue
		}

		// 尝试从任务通道获取任务
		task, ok := <-dq.tasks
		if !ok {
			// 通道已关闭，退出工作协程
			return
		}
		// 执行下载任务
		task.Download()
		time.Sleep(10 * time.Millisecond) // 等待10毫秒，防止CPU占用过高，也给其他协程流出数据库写入时间
	}
}

func (dq *DQ) IsRunning() bool {
	dq.mutex.RLock()
	defer dq.mutex.RUnlock()
	return dq.running
}

// RestartGlobalDownloadQueue 重启全局下载队列
func RestartGlobalDownloadQueue() {
	if GlobalDownloadQueue != nil {
		GlobalDownloadQueue.Restart()
		helpers.AppLogger.Info("全局下载队列已重启")
	} else {
		helpers.AppLogger.Error("全局下载队列未初始化")
	}
}

// UpdateGlobalDownloadQueueConcurrency 更新全局下载队列的并发数
func UpdateGlobalDownloadQueueConcurrency(newConcurrency int) {
	if GlobalDownloadQueue != nil {
		GlobalDownloadQueue.UpdateConcurrency(newConcurrency)
		helpers.AppLogger.Infof("全局下载队列并发数已更新为 %d", newConcurrency)
	} else {
		helpers.AppLogger.Error("全局下载队列未初始化")
	}
}

// GetGlobalDownloadQueueConcurrency 获取全局下载队列的并发数
func GetGlobalDownloadQueueConcurrency() int {
	if GlobalDownloadQueue != nil {
		return GlobalDownloadQueue.GetConcurrency()
	}
	return 0
}
