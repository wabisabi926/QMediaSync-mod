package requests

import (
	"qmediasync/internal/models"
	"qmediasync/internal/validation"
)

// UpdateThreadsRequest 更新线程配置请求。
type UpdateThreadsRequest struct {
	DownloadThreads    int `form:"download_threads" json:"download_threads" binding:"required"`
	FileDetailThreads  int `form:"file_detail_threads" json:"file_detail_threads" binding:"required"`
	OpenlistQPS        int `form:"openlist_qps" json:"openlist_qps" binding:"required"`
	OpenlistRetry      int `form:"openlist_retry" json:"openlist_retry" binding:"required"`
	OpenlistRetryDelay int `form:"openlist_retry_delay" json:"openlist_retry_delay" binding:"required"`
	FileListPageSize   int `form:"file_list_page_size" json:"file_list_page_size" binding:"required"`
}

// Validate 校验线程配置请求。
func (r UpdateThreadsRequest) Validate() error {
	if err := validation.RangeInt("download_threads", r.DownloadThreads, 1, 10); err != nil {
		return err
	}
	if err := validation.RangeInt("file_detail_threads", r.FileDetailThreads, 2, 10); err != nil {
		return err
	}
	if err := validation.RangeInt("openlist_qps", r.OpenlistQPS, 2, 10); err != nil {
		return err
	}
	if err := validation.RangeInt("openlist_retry", r.OpenlistRetry, 1, 10); err != nil {
		return err
	}
	if err := validation.RangeInt("openlist_retry_delay", r.OpenlistRetryDelay, 30, 3600); err != nil {
		return err
	}
	return validation.RangeInt("file_list_page_size", r.FileListPageSize, 100, 1150)
}

// ToModel 转换为线程配置模型。
func (r UpdateThreadsRequest) ToModel() models.SettingThreads {
	return models.SettingThreads{
		DownloadThreads:    r.DownloadThreads,
		FileDetailThreads:  r.FileDetailThreads,
		OpenlistQPS:        r.OpenlistQPS,
		OpenlistRetry:      r.OpenlistRetry,
		OpenlistRetryDelay: r.OpenlistRetryDelay,
		FileListPageSize:   r.FileListPageSize,
	}
}

// UpdateStrmConfigRequest 更新 STRM 配置请求。
type UpdateStrmConfigRequest struct {
	LocalProxy     int      `form:"local_proxy" json:"local_proxy"`
	StrmBaseURL    string   `form:"strm_base_url" json:"strm_base_url" binding:"required"`
	Cron           string   `form:"cron" json:"cron" binding:"required"`
	MinVideoSize   int64    `form:"min_video_size" json:"min_video_size"`
	VideoExtArr    []string `json:"video_ext_arr"`
	MetaExtArr     []string `form:"meta_ext_arr" json:"meta_ext_arr"`
	ExcludeNameArr []string `form:"exclude_name_arr" json:"exclude_name_arr"`
	UploadMeta     int      `form:"upload_meta" json:"upload_meta"`
	DownloadMeta   int      `form:"download_meta" json:"download_meta"`
	DeleteDir      int      `form:"delete_dir" json:"delete_dir"`
	AddPath        int      `form:"add_path" json:"add_path"`
	CheckMetaMtime int      `form:"check_meta_mtime" json:"check_meta_mtime"`
}

// Validate 校验 STRM 配置请求。
func (r UpdateStrmConfigRequest) Validate() error {
	if err := validation.HTTPURL("strm_base_url", r.StrmBaseURL, false); err != nil {
		return err
	}
	if err := validation.Cron("cron", r.Cron, false); err != nil {
		return err
	}
	if err := validation.RangeInt64("min_video_size", r.MinVideoSize, 0, 9223372036854775807); err != nil {
		return err
	}
	if err := validation.ExtList("video_ext_arr", r.VideoExtArr, false); err != nil {
		return err
	}
	if err := validation.ExtList("meta_ext_arr", r.MetaExtArr, false); err != nil {
		return err
	}
	if err := validation.OneOfInt("local_proxy", r.LocalProxy, []int{0, 1}); err != nil {
		return err
	}
	if err := validation.OneOfInt("upload_meta", r.UploadMeta, []int{0, 1, 2}); err != nil {
		return err
	}
	if err := validation.OneOfInt("download_meta", r.DownloadMeta, []int{0, 1}); err != nil {
		return err
	}
	if err := validation.OneOfInt("delete_dir", r.DeleteDir, []int{0, 1}); err != nil {
		return err
	}
	if err := validation.OneOfInt("add_path", r.AddPath, []int{1, 2, 3}); err != nil {
		return err
	}
	return validation.OneOfInt("check_meta_mtime", r.CheckMetaMtime, []int{0, 1})
}

// ToModel 转换为 STRM 配置模型。
func (r UpdateStrmConfigRequest) ToModel() models.SettingStrm {
	return models.SettingStrm{
		LocalProxy:     r.LocalProxy,
		StrmBaseUrl:    r.StrmBaseURL,
		Cron:           r.Cron,
		MinVideoSize:   r.MinVideoSize,
		VideoExtArr:    r.VideoExtArr,
		MetaExtArr:     r.MetaExtArr,
		ExcludeNameArr: r.ExcludeNameArr,
		UploadMeta:     r.UploadMeta,
		DownloadMeta:   r.DownloadMeta,
		DeleteDir:      r.DeleteDir,
		AddPath:        r.AddPath,
		CheckMetaMtime: r.CheckMetaMtime,
	}
}
