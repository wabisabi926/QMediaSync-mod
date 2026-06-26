package requests

import (
	"qmediasync/internal/models"
	"qmediasync/internal/validation"
)

// SaveScrapePathRequest 保存刮削路径请求。
type SaveScrapePathRequest struct {
	ID                    uint              `json:"id" form:"id"`
	AccountID             uint              `json:"account_id" form:"account_id"`
	SourceType            models.SourceType `json:"source_type" form:"source_type"`
	MediaType             models.MediaType  `json:"media_type" form:"media_type"`
	SourcePath            string            `json:"source_path" form:"source_path"`
	SourcePathID          string            `json:"source_path_id" form:"source_path_id"`
	DestPath              string            `json:"dest_path" form:"dest_path"`
	DestPathID            string            `json:"dest_path_id" form:"dest_path_id"`
	ScrapeType            models.ScrapeType `json:"scrape_type" form:"scrape_type"`
	RenameType            models.RenameType `json:"rename_type" form:"rename_type"`
	FolderNameTemplate    string            `json:"folder_name_template" form:"folder_name_template"`
	FileNameTemplate      string            `json:"file_name_template" form:"file_name_template"`
	DeleteKeyword         []string          `json:"delete_keyword" form:"delete_keyword"`
	EnableCategory        bool              `json:"enable_category" form:"enable_category"`
	VideoExtList          []string          `json:"video_ext_list" form:"video_ext_list"`
	MinVideoFileSize      int64             `json:"min_video_file_size" form:"min_video_file_size"`
	ExcludeNoImageActor   bool              `json:"exclude_no_image_actor" form:"exclude_no_image_actor"`
	EnableAi              models.AiAction   `json:"enable_ai" form:"enable_ai"`
	AiPrompt              string            `json:"ai_prompt" form:"ai_prompt"`
	ForceDeleteSourcePath bool              `json:"force_delete_source_path" form:"force_delete_source_path"`
	EnableCron            bool              `json:"enable_cron" form:"enable_cron"`
	CronExpression        string            `json:"cron_expression" form:"cron_expression"`
	EnableFanartTv        bool              `json:"enable_fanart_tv" form:"enable_fanart_tv"`
	MaxThreads            int               `json:"max_threads" form:"max_threads"`
}

// Validate 校验刮削路径请求。
func (r SaveScrapePathRequest) Validate() error {
	sourceValues := []string{
		string(models.SourceTypeLocal),
		string(models.SourceType115),
		string(models.SourceType123),
		string(models.SourceTypeOpenList),
		string(models.SourceTypeBaiduPan),
	}
	if err := validation.OneOfString("source_type", string(r.SourceType), sourceValues); err != nil {
		return err
	}
	if r.SourceType != models.SourceTypeLocal && r.AccountID == 0 {
		return validation.New("account_id", "非本地来源必须选择账号")
	}
	if err := validation.OneOfString("media_type", string(r.MediaType), []string{
		string(models.MediaTypeMovie),
		string(models.MediaTypeTvShow),
		string(models.MediaTypeOther),
	}); err != nil {
		return err
	}
	if err := validation.OneOfString("scrape_type", string(r.ScrapeType), []string{
		string(models.ScrapeTypeOnly),
		string(models.ScrapeTypeScrapeAndRename),
		string(models.ScrapeTypeOnlyRename),
	}); err != nil {
		return err
	}
	if err := validation.OneOfString("rename_type", string(r.RenameType), []string{
		string(models.RenameTypeMove),
		string(models.RenameTypeCopy),
		string(models.RenameTypeSoftSymlink),
		string(models.RenameTypeHardSymlink),
	}); err != nil {
		return err
	}
	if r.SourceType != models.SourceTypeLocal && r.RenameType != models.RenameTypeMove {
		return validation.New("rename_type", "非本地来源只支持移动重命名")
	}
	if err := validation.NonBlank("source_path", r.SourcePath); err != nil {
		return err
	}
	if err := validation.NonBlank("dest_path", r.DestPath); err != nil {
		return err
	}
	if err := validation.ExtList("video_ext_list", r.VideoExtList, true); err != nil {
		return err
	}
	if r.MinVideoFileSize < 0 {
		return validation.New("min_video_file_size", "不能小于 0")
	}
	maxThreads := 5
	if r.SourceType == models.SourceTypeLocal {
		maxThreads = 20
	}
	if err := validation.RangeInt("max_threads", r.MaxThreads, 1, maxThreads); err != nil {
		return err
	}
	if r.EnableCron {
		return validation.Cron("cron_expression", r.CronExpression, false)
	}
	return validation.Cron("cron_expression", r.CronExpression, true)
}

// ToModel 转换为刮削路径模型。
func (r SaveScrapePathRequest) ToModel() models.ScrapePath {
	return models.ScrapePath{
		BaseModel:             models.BaseModel{ID: r.ID},
		AccountId:             r.AccountID,
		SourceType:            r.SourceType,
		MediaType:             r.MediaType,
		SourcePath:            r.SourcePath,
		SourcePathId:          r.SourcePathID,
		DestPath:              r.DestPath,
		DestPathId:            r.DestPathID,
		ScrapeType:            r.ScrapeType,
		RenameType:            r.RenameType,
		FolderNameTemplate:    r.FolderNameTemplate,
		FileNameTemplate:      r.FileNameTemplate,
		DeleteKeyword:         r.DeleteKeyword,
		EnableCategory:        r.EnableCategory,
		VideoExtList:          r.VideoExtList,
		MinVideoFileSize:      r.MinVideoFileSize,
		ExcludeNoImageActor:   r.ExcludeNoImageActor,
		EnableAi:              r.EnableAi,
		AiPrompt:              r.AiPrompt,
		ForceDeleteSourcePath: r.ForceDeleteSourcePath,
		EnableCron:            r.EnableCron,
		CronExpression:        r.CronExpression,
		EnableFanartTv:        r.EnableFanartTv,
		MaxThreads:            r.MaxThreads,
	}
}
