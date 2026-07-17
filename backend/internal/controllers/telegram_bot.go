package controllers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"qmediasync/internal/helpers"
	"qmediasync/internal/models"
	"qmediasync/internal/notificationmanager"
	"qmediasync/internal/synccron"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// checkAndExtractSingleParam 检查并提取单个任务参数
// args： 参数列表
// 返回错误信息（如果参数格式错误）和提取的任务 ID（如果成功）
// 如果没有参数或参数为空，返回空错误和 0。
func checkAndExtractSingleParam(args []string) (string, uint) {
	if len(args) > 0 && args[0] != "" {
		param := args[0]
		// 检查参数是否以 # 开头且长度大于 1。
		if !(len(param) > 1 && param[0] == '#') {
			return "❌ 参数格式错误，请使用 #数字 格式", 0
		}
		// 尝试将参数转换为 uint。
		numStr := param[1:]
		id, parseErr := strconv.ParseUint(numStr, 10, 32)
		if parseErr != nil {
			return "❌ 参数格式错误，请使用 #数字 格式", 0
		}
		return "", uint(id)
	}
	return "", 0
}

// checkAndExtractMoreParam 检查并提取多个任务参数
// args： 参数列表
// 返回错误信息（如果参数格式错误）和提取的任务 ID 列表（如果成功）。
func checkAndExtractMoreParam(args []string) (string, []uint) {
	var taskIDs []uint
	for _, arg := range args {
		if arg != "" {
			// 检查参数是否以 # 开头且长度大于 1。
			if !(len(arg) > 1 && arg[0] == '#') {
				return "❌ 参数格式错误，请使用 #数字 #数字 格式", nil
			}
			// 尝试将参数转换为 uint。
			numStr := arg[1:]
			id, parseErr := strconv.ParseUint(numStr, 10, 32)
			if parseErr != nil {
				return "❌ 参数格式错误，请使用 #数字 #数字 格式", nil
			}
			taskIDs = append(taskIDs, uint(id))
		}
	}
	return "", taskIDs
}

// runStrmTask 执行 STRM 同步任务
// args： 可选参数，传入同步目录 ID 时只同步指定目录
// isFullSync： 是否执行全量同步
func runStrmTask(taskID uint, isFullSync bool) string {
	go runStrmTaskSync(taskID, isFullSync)
	// 返回开始执行的消息
	if isFullSync {
		return "🔄 开始执行全量 STRM 同步"
	}
	return "🔄 开始执行增量 STRM 同步"
}

func runStrmTaskSync(taskID uint, isFullSync bool) {
	// 先返回开始执行的消息
	taskIDs := []uint{}
	var title, content string

	// 设置通知信息
	if isFullSync {
		title = "✅ 全量 STRM 同步完成"
		content = "所有全量 STRM 同步任务已执行完毕"
	} else {
		title = "✅ 增量 STRM 同步完成"
		content = "所有增量 STRM 同步任务已执行完毕"
	}

	// 检查是否传入了目录 ID。
	if taskID > 0 {
		// 获取指定同步目录
		syncPath := models.GetSyncPathById(taskID)
		if syncPath != nil {
			// 如果是全量同步，设置标志
			if isFullSync {
				syncPath.SetIsFullSync(true)
			}
			// 同步指定目录
			taskObj := &synccron.NewSyncTask{
				ID:           syncPath.ID,
				SourcePath:   "",
				SourcePathId: "",
				TargetPath:   "",
				AccountId:    syncPath.AccountId,
				IsFile:       false,
				TaskType:     synccron.SyncTaskTypeStrm,
				SourceType:   syncPath.SourceType,
			}
			synccron.AddNewSyncTask(taskObj)
			taskIDs = []uint{taskID}
			// 设置通知内容
			if isFullSync {
				content = "目录：" + syncPath.RemotePath + "，全量 STRM 同步任务已执行完毕"
			} else {
				content = "目录：" + syncPath.RemotePath + "，增量 STRM 同步任务已执行完毕"
			}
		}

	} else {
		// 获取所有同步目录
		allSyncPaths, _ := models.GetSyncPathList(1, 10000000, false, "")
		for _, syncPath := range allSyncPaths {
			// 全量同步时设置标志
			if isFullSync {
				syncPath.SetIsFullSync(true)
			}
			// 同步目录
			taskObj := &synccron.NewSyncTask{
				ID:           syncPath.ID,
				SourcePath:   "",
				SourcePathId: "",
				TargetPath:   "",
				AccountId:    syncPath.AccountId,
				IsFile:       false,
				TaskType:     synccron.SyncTaskTypeStrm,
				SourceType:   syncPath.SourceType,
			}
			synccron.AddNewSyncTask(taskObj)
			taskIDs = append(taskIDs, syncPath.ID)
		}
		// 设置通知内容
		if isFullSync {
			content = "目录：全部，全量 STRM 同步任务已执行完毕"
		} else {
			content = "目录：全部，增量 STRM 同步任务已执行完毕"
		}
	}

	// 等待所有任务执行完成
	time.Sleep(2 * time.Second) // 等待任务队列初始化

	// 监控任务的状态
	waitForTasksCompletion(taskIDs, synccron.SyncTaskTypeStrm)

	// 所有任务执行完成，发送通知
	ctx := context.Background()
	notif := &models.Notification{
		Type:      models.SystemAlert,
		Title:     title,
		Content:   content,
		Timestamp: time.Now(),
		Priority:  models.NormalPriority,
	}
	if notificationmanager.GlobalEnhancedNotificationManager != nil {
		notificationmanager.GlobalEnhancedNotificationManager.SendNotification(ctx, notif)
	}
}

// SyncStrmInc 执行增量 STRM 同步并在完成后发送通知
// args： 可选参数，传入同步目录 ID 时只同步指定目录
func SyncStrmInc(args []string) helpers.CommandResponse {
	if errMsg, _ := checkAndExtractSingleParam(args); errMsg != "" {
		return helpers.CommandResponse{Text: errMsg}
	}
	_, taskID := checkAndExtractSingleParam(args)
	return helpers.CommandResponse{Text: runStrmTask(taskID, false)}
}

// SyncStrnFull 执行全量 STRM 同步并在完成后发送通知
// args： 可选参数，传入同步目录 ID 时只同步指定目录
func SyncStrnFull(args []string) helpers.CommandResponse {
	if errMsg, _ := checkAndExtractSingleParam(args); errMsg != "" {
		return helpers.CommandResponse{Text: errMsg}
	}
	_, taskID := checkAndExtractSingleParam(args)
	return helpers.CommandResponse{Text: runStrmTask(taskID, true)}
}

// waitForTasksCompletion 等待指定任务完成
func waitForTasksCompletion(taskIDs []uint, taskType synccron.SyncTaskType) {
	if len(taskIDs) == 0 {
		return
	}
	allCompleted := false
	for !allCompleted {
		time.Sleep(5 * time.Second)
		allCompleted = true
		for _, taskID := range taskIDs {
			status := synccron.CheckNewTaskStatus(taskID, taskType)
			if status == synccron.TaskStatusWaiting || status == synccron.TaskStatusRunning {
				allCompleted = false
				break
			}
		}
	}
}

// ParseStrmPathArgs 解析 get_strm_path 命令的参数
func ParseStrmPathArgs(args []string) (int, int) {
	page := 1
	pageSize := 20

	// 解析参数
	if len(args) >= 1 && args[0] != "" {
		if num, err := strconv.Atoi(args[0][1:]); err == nil && num > 0 {
			page = num
		}
	}

	if len(args) >= 2 && args[1] != "" {
		if num, err := strconv.Atoi(args[1][1:]); err == nil && num > 0 {
			pageSize = num
		}
	}

	return page, pageSize
}

// getStrmPath 获取同步路径列表
// args： 可选参数，传入页码和每页数量，格式为 #页码 #每页数量
func getStrmPath(args []string) helpers.CommandResponse {
	page, pageSize := ParseStrmPathArgs(args)

	// 获取同步路径列表
	syncPaths, total := models.GetSyncPathList(page, pageSize, false, "")

	// 格式化输出
	result := "📋 STRM 同步路径列表\n"
	result += fmt.Sprintf("第 %d 页，共 %d 条记录\n\n", page, total)

	for _, sp := range syncPaths {
		status := synccron.CheckNewTaskStatus(sp.ID, synccron.SyncTaskTypeStrm)
		statusStr := "⏸️ 空闲"
		switch status {
		case synccron.TaskStatusRunning:
			statusStr = "🔄 运行中"
		case synccron.TaskStatusWaiting:
			statusStr = "⏳ 等待中"
		}

		result += fmt.Sprintf("  ID：#%d\n", sp.ID)
		result += fmt.Sprintf("  原始路径：%s\n", sp.RemotePath)
		result += fmt.Sprintf("  目标路径：%s\n", sp.LocalPath)
		result += fmt.Sprintf("  状态：%s\n", statusStr)
		result += fmt.Sprintf("  来源：%s\n", sp.SourceType)
		result += fmt.Sprintf("  最后同步：%s\n\n", time.Unix(sp.UpdatedAt, 0).Format("2006-01-02 15:04"))
	}

	// 构建内联键盘
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, sp := range syncPaths {
		row := tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("#%d（增量同步）", sp.ID),
				fmt.Sprintf("strm_inc #%d", sp.ID),
			),
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("#%d（全量同步）", sp.ID),
				fmt.Sprintf("strm_sync #%d", sp.ID),
			),
		)
		rows = append(rows, row)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	return helpers.CommandResponse{
		Text:        result,
		ReplyMarkup: keyboard,
	}
}

// ParseScrapePathArgs 解析 get_scrape_path 命令的参数
func ParseScrapePathArgs(args []string) string {
	sourceType := ""

	// 解析参数
	if len(args) >= 1 && args[0] != "" {
		sourceType = args[0][1:]
	}

	return sourceType
}

func StartListenTelegramBot() {
	mgr := notificationmanager.GlobalEnhancedNotificationManager

	myCommands := map[string]func([]string) helpers.CommandResponse{
		"strm_inc":      SyncStrmInc,
		"strm_sync":     SyncStrnFull,
		"get_strm_path": getStrmPath,
	}

	mgr.RegisterTelegramCommands(myCommands)
	mgr.StartAll()
}
