package controllers

import (
	"Q115-STRM/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// UploadList 获取上传队列列表
// @Summary 获取上传队列
// @Description 按状态分页获取上传队列任务列表
// @Tags 队列管理
// @Accept json
// @Produce json
// @Param status query string false "任务状态"
// @Param page query integer false "页码，默认1"
// @Param page_size query integer false "每页数量，默认100"
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /upload/queue [get]
// @Security JwtAuth
// @Security ApiKeyAuth
func UploadList(ctx *gin.Context) {
	type uploadListReq struct {
		Status   models.UploadStatus `json:"status" form:"status"`
		Page     int                 `json:"page" form:"page"`
		PageSize int                 `json:"page_size" form:"page_size"`
	}
	var req uploadListReq
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
		return
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 100
	}
	// 从请求中获取文件列表
	// 从model/upload.go中查询上传队列列表
	type uploadQueueResp struct {
		Total     int                    `json:"total"`
		Uploading int                    `json:"uploading"`
		List      []*models.DbUploadTask `json:"list"`
	}
	// 从请求中获取文件列表
	// 从model/upload.go中查询上传队列列表
	uploadList, total := models.GetUploadTaskList(req.Status, req.Page, req.PageSize)
	ctx.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "上传队列列表查询成功", Data: uploadQueueResp{
		Total:     int(total),
		Uploading: int(models.GetUploadingCount()),
		List:      uploadList,
	}})
}

// ClearPendingUploadTasks 清除上传队列中未开始的任务
// @Summary 清空待上传任务
// @Description 清除上传队列中所有未开始的任务
// @Tags 队列管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /upload/queue/clear-pending [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func ClearPendingUploadTasks(ctx *gin.Context) {
	// 调用全局上传队列的ClearPendingTasks方法
	err := models.ClearPendingUploadTasks()
	if err != nil {
		ctx.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "清除待上传任务失败", Data: nil})
		return
	}

	// 返回结果
	ctx.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "成功清除待上传任务", Data: nil})
}

// ClearUploadSuccessAndFailedTasks 清除上传队列中成功和失败的任务
// @Summary 清空已完成/失败的上传任务
// @Description 删除上传队列中已成功和失败的任务
// @Tags 队列管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /upload/queue/clear-success-failed [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func ClearUploadSuccessAndFailedTasks(ctx *gin.Context) {
	err := models.ClearUploadSuccessAndFailed()
	if err != nil {
		ctx.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "删除成功和失败任务失败", Data: nil})
		return
	}

	ctx.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "成功删除成功和失败任务", Data: nil})
}

// RetryFailedUploadTasks 重试所有失败的上传任务
// @Summary 重试失败的上传任务
// @Description 将所有失败的上传任务状态改为等待中，会自动触发重试
// @Tags 队列管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /upload/queue/retry-failed [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func RetryFailedUploadTasks(ctx *gin.Context) {
	err := models.RetryFailedUploadTasks()
	if err != nil {
		ctx.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "重试失败任务失败", Data: nil})
		return
	}

	ctx.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "重试失败任务成功", Data: nil})
}

// StartUploadQueue 启动上传队列
// @Summary 启动上传队列
// @Description 启动或恢复上传队列执行
// @Tags 队列管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /upload/queue/start [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func StartUploadQueue(ctx *gin.Context) {
	// 调用全局上传队列的Start方法
	models.GlobalUploadQueue.Restart()

	// 返回结果
	ctx.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "下载队列已启动", Data: nil})
}

// StopUploadQueue 停止上传队列
// @Summary 停止上传队列
// @Description 停止上传队列执行
// @Tags 队列管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /upload/queue/stop [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func StopUploadQueue(ctx *gin.Context) {
	// 调用全局上传队列的Stop方法
	models.GlobalUploadQueue.Stop()

	// 返回结果
	ctx.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "上传队列已停止", Data: nil})
}

// UploadQueueStatus 查询上传队列状态
// @Summary 查询上传队列状态
// @Description 获取上传队列当前运行状态
// @Tags 队列管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /upload/queue/status [get]
// @Security JwtAuth
// @Security ApiKeyAuth
func UploadQueueStatus(ctx *gin.Context) {
	// 调用全局上传队列的GetStatus方法
	status := models.GlobalUploadQueue.IsRunning()

	// 返回结果
	ctx.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "下载队列状态查询成功", Data: status})
}

// DownloadList 获取下载队列列表
// @Summary 获取下载队列
// @Description 按状态分页获取下载队列任务列表
// @Tags 队列管理
// @Accept json
// @Produce json
// @Param status query string false "任务状态"
// @Param page query integer false "页码，默认1"
// @Param page_size query integer false "每页数量，默认100"
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /download/queue [get]
// @Security JwtAuth
// @Security ApiKeyAuth
func DownloadList(ctx *gin.Context) {
	type downloadListReq struct {
		Status   models.DownloadStatus `json:"status" form:"status"`
		Page     int                   `json:"page" form:"page"`
		PageSize int                   `json:"page_size" form:"page_size"`
	}
	type downloadQueueResp struct {
		Total       int64                    `json:"total"`
		Downloading int64                    `json:"downloading"`
		List        []*models.DbDownloadTask `json:"list"`
	}
	var req downloadListReq
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "请求参数错误", Data: nil})
		return
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 100
	}
	// 从请求中获取文件列表
	// 从model/download.go中查询下载队列列表
	downloadList, total := models.GetDownloadTaskList(req.Status, req.Page, req.PageSize)
	ctx.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "下载队列列表查询成功", Data: downloadQueueResp{
		Total:       total,
		Downloading: models.GetDownloadingCount(),
		List:        downloadList,
	}})
}

// ClearPendingDownloadTasks 清除下载队列中未开始的任务
// @Summary 清空待下载任务
// @Description 清除下载队列中所有未开始的任务
// @Tags 队列管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /download/queue/clear-pending [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func ClearPendingDownloadTasks(ctx *gin.Context) {
	// 调用全局下载队列的ClearPendingTasks方法
	err := models.ClearDownloadPendingTasks()
	if err != nil {
		ctx.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "清除下载任务失败", Data: nil})
		return
	}

	// 返回结果
	ctx.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "成功清除下载任务", Data: nil})
}

// StartDownloadQueue 启动下载队列
// @Summary 启动下载队列
// @Description 启动或恢复下载队列执行
// @Tags 队列管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /download/queue/start [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func StartDownloadQueue(ctx *gin.Context) {
	// 调用全局下载队列的Start方法
	models.GlobalDownloadQueue.Restart()

	// 返回结果
	ctx.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "下载队列已启动", Data: nil})
}

// StopDownloadQueue 停止下载队列
// @Summary 停止下载队列
// @Description 停止下载队列执行
// @Tags 队列管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /download/queue/stop [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func StopDownloadQueue(ctx *gin.Context) {
	// 调用全局下载队列的Stop方法
	models.GlobalDownloadQueue.Stop()

	// 返回结果
	ctx.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "下载队列已停止", Data: nil})
}

// DownloadQueueStatus 查询下载队列状态
// @Summary 查询下载队列状态
// @Description 获取下载队列当前运行状态
// @Tags 队列管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /download/queue/status [get]
// @Security JwtAuth
// @Security ApiKeyAuth
func DownloadQueueStatus(ctx *gin.Context) {
	// 调用全局下载队列的GetStatus方法
	status := models.GlobalDownloadQueue.IsRunning()

	// 返回结果
	ctx.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "下载队列状态查询成功", Data: status})
}

// ClearDownloadSuccessAndFailedTasks 清除下载队列中成功和失败的任务
// @Summary 清空已完成/失败的下载任务
// @Description 删除下载队列中已成功和失败的任务
// @Tags 队列管理
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 200 {object} object
// @Router /download/queue/clear-success-failed [post]
// @Security JwtAuth
// @Security ApiKeyAuth
func ClearDownloadSuccessAndFailedTasks(ctx *gin.Context) {
	// 调用全局下载队列的DeleteSuccessAndFailed方法
	err := models.ClearDownloadSuccessAndFailed()
	if err != nil {
		ctx.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "删除成功和失败任务失败", Data: nil})
		return
	}

	// 返回结果
	ctx.JSON(http.StatusOK, APIResponse[any]{Code: Success, Message: "成功删除成功和失败任务", Data: nil})
}
