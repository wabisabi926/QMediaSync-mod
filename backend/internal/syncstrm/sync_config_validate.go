package syncstrm

import (
	"Q115-STRM/internal/models"
	"path/filepath"
	"slices"
	"strings"
)

// 允许上传的目录，网盘中哪怕没有这些目录，也可以上传这些目录下的文件到网盘
var uploadDirNames = []string{
	"extrafanart",
	"exfanarts",
	"extrafanarts",
	"extras",
	"specials",
	"shorts",
	"scenes",
	"featurettes",
	"behind the scenes",
	"trailers",
	"interviews",
	"subs",
}

type SyncStrmConfig struct {
	StrmBaseUrl           string                        `json:"strm_base_url"`             // 视频文件URL基础路径
	MinVideoSize          int64                         `json:"min_video_size"`            // 视频文件最小大小，单位为MB
	EnableDownloadMeta    int64                         `json:"enable_download_meta"`      // 是否下载元数据文件，0为不下载，1为下载
	NetNotFoundFileAction models.SyncTreeItemMetaAction `json:"net_not_found_file_action"` // 网盘文件不存在时的操作，0为忽略，1为上传，2-删除
	VideoExt              []string                      `json:"video_ext"`                 // 视频文件扩展名列表
	MetaExt               []string                      `json:"meta_ext"`                  // 元数据文件扩展名列表
	ExcludeNames          []string                      `json:"exclude_names"`             // 排除的文件名列表
	StrmUrlNeedPath       int                           `json:"strm_url_need_path"`        // 视频文件URL是否需要路径，2为不需要，1为需要
	DelEmptyLocalDir      bool                          `json:"del_empty_local_dir"`       // 是否删除本地空目录
	CheckMetaMtime        int                           `json:"check_meta_mtime"`          // 是否检查元数据文件修改时间，默认0， 如果1，网盘新则下载，网盘旧就上传（UploadMeta=1时）
}

func (s *SyncStrm) ValidFile(file *SyncFileCache) bool {
	// 排除在上一步已经做了
	// // 检查文件是否被排除
	// if s.IsExcludeName(filepath.Base(file.FileName)) {
	// 	s.Sync.Logger.Warnf("文件 %s 被排除，跳过它和旗下所有内容", file.FileName)
	// 	return false
	// }
	// 如果是文件，则进行预处理，然后插入临时表
	file.IsVideo = s.IsValidVideoExt(file.FileName)
	file.IsMeta = s.IsValidMetaExt(file.FileName)
	if !file.IsVideo && !file.IsMeta {
		s.Sync.Logger.Infof("文件 %s 不是视频或元数据，跳过", file.FileName)
		return false
	}
	maxSize := s.GetMinVideoSize()
	if file.IsVideo && file.FileSize < maxSize {
		s.Sync.Logger.Infof("视频文件%s大小%d小于%d最小要求，不需要处理", file.FileName, file.FileSize, maxSize)
		return false
	}
	if file.IsMeta && !s.EnableDownloadMeta() {
		// 如果是元数据文件且设置为不下载，则跳过检查（代表着不上传）
		// s.Sync.Logger.Infof("网盘元数据文件 %s 由于关闭了元数据下载所以不需要处理", file.FileName)
		return false
	}
	return true
}
func (s *SyncStrm) GetMinVideoSize() int64 {
	if s.Config.MinVideoSize > 0 {
		return s.Config.MinVideoSize * 1024 * 1024
	}
	return 0
}

func (s *SyncStrm) EnableDownloadMeta() bool {
	if s.Config.EnableDownloadMeta > 0 {
		return true
	}
	return false
}

func (s *SyncStrm) IsValidVideoExt(filename string) bool {
	ext := filepath.Ext(filename)
	ext = strings.ToLower(ext)
	if slices.Contains(s.Config.VideoExt, ext) {
		return true
	}
	return false
}

func (s *SyncStrm) IsValidMetaExt(filename string) bool {
	ext := filepath.Ext(filename)
	ext = strings.ToLower(ext)
	if slices.Contains(s.Config.MetaExt, ext) {
		return true
	}
	return false
}

func (s *SyncStrm) IsExcludeName(filename string) bool {
	if len(s.Config.ExcludeNames) == 0 {
		return false
	}
	filename = strings.ToLower(filename)
	return slices.Contains(s.Config.ExcludeNames, strings.ToLower(filename))
}

func (s *SyncStrm) IsExcludePath(path string) bool {
	// 分隔路径
	pathParts := strings.Split(path, "/")
	for _, part := range pathParts {
		if s.IsExcludeName(part) {
			return true
		}
	}
	return false
}
