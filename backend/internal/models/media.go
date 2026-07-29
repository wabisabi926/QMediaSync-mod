package models

import (
	"encoding/json"

	"qmediasync/internal/db"
	"qmediasync/internal/helpers"
)

type MediaStatus string

const (
	MediaStatusScraped   MediaStatus = "scraped" // 已刮削
	MediaStatusUnScraped MediaStatus = "scanned" // 待刮削
	MediaStatusRenamed   MediaStatus = "renamed" // 已重命名
)

type MediaType string

const (
	MediaTypeMovie   MediaType = "movie"
	MediaTypeTvShow  MediaType = "tvshow"
	MediaTypeUnknown MediaType = "unknown"
)

type MediaMetaFiles struct {
	FileName     string `json:"file_name"`
	FilePath     string `json:"file_path"`
	IsDefault    bool   `json:"is_default"`
	Language     string `json:"language"`
	Forced       bool   `json:"forced"`
	ProviderName string `json:"provider_name"`
}

// 电影或电视剧，刮削后的信息
// 每一个刮削过的视频都对应一条 Media 记录
type Media struct {
	BaseModel
	TmdbId              int64              `json:"tmdb_id" gorm:"index:tmdbid"`              // TMDB ID
	ImdbId              string             `json:"imdb_id"`                                  // IMDb ID
	Name                string             `json:"name" gorm:"index:nameyear"`               // TMDB 名称
	Year                int                `json:"year" gorm:"index:nameyear"`               // 年份
	OriginalName        string             `json:"original_title"`                           // 原始标题
	MediaType           MediaType          `json:"media_type"`                               // 媒体类型：movie-电影 tvshow-电视剧
	ReleaseDate         string             `json:"release_date"`                             // 上映时间，剧集为首播时间
	Actors              []helpers.Actor    `json:"actors" gorm:"-"`                          // 演员列表
	ActorsJson          string             `json:"-" gorm:"type:text"`                       // 演员列表 JSON 字符串
	Director            []helpers.Director `json:"director" gorm:"-"`                        // 导演列表
	DirectorJson        string             `json:"-" gorm:"type:text"`                       // 导演列表 JSON 字符串
	Overview            string             `json:"overview"`                                 // 媒体描述
	Tagline             string             `json:"tagline"`                                  // 媒体标语
	PosterPath          string             `json:"poster_path"`                              // 海报路径，poster.jpg 竖版，推荐 1000x1500 像素，比例：2:3
	BackdropPath        string             `json:"backdrop_path"`                            // 背景路径，生成 fanart.jpg、backdrop.jpg、background.jpg 三个横版文件，推荐 1920x1080 像素，比例：16:9
	LogoPath            string             `json:"logo_path"`                                // logo 路径，clearlogo.jpg
	ThumbPath           string             `json:"thumb_path"`                               // 缩略图路径，thumb.jpg 推荐 400x300 像素，比例：4:3 // 暂时不用
	LandscapePath       string             `json:"landscape_path"`                           // 横屏海报路径，landscape.jpg 尺寸：约 1000x562 比例：16:9 // 暂时不用
	BannerPath          string             `json:"banner_path"`                              // 超宽横幅路径，banner.jpg 尺寸：约 1000x185 像素 5.4:1 (超宽) 暂时不用
	VoteAverage         float64            `json:"vote_average"`                             // 投票平均分
	VoteCount           int64              `json:"vote_count"`                               // 投票数
	OriginalLanguage    string             `json:"original_language"`                        // 原始语言
	OriginCountry       []string           `json:"origin_country" gorm:"-"`                  // 原始国家
	OriginalCountryJson string             `json:"-"`                                        // 原始国家 JSON 字符串
	Runtime             int64              `json:"runtime"`                                  // 运行时间，单位：分钟
	LastAirDate         string             `json:"last_air_date"`                            // 最后一集播出时间
	NumberOfEpisodes    int                `json:"number_of_episodes"`                       // 剧集总数
	NumberOfSeasons     int                `json:"number_of_seasons"`                        // 季数
	Num                 string             `json:"num"`                                      // 番号
	MpaaRating          string             `json:"mpaa_rating"`                              // MPAA 分级
	Path                string             `json:"path"`                                     // 刮削整理后的电影或者电视剧的路径
	PathId              string             `json:"path_id"`                                  // 刮削整理后的电影或者电视剧的路径 ID
	VideoFileName       string             `json:"video_file_name"`                          // 刮削整理后的电影或者电视剧的视频文件名
	VideoFileId         string             `json:"video_file_id"`                            // 刮削整理后的电影或者电视剧的视频文件 ID，完整路径或者网盘文件 ID
	VideoPickCode       string             `json:"video_pick_code"`                          // 115 PickCode 或百度网盘 fs_id
	VideoOpenListSign   string             `json:"video_open_list_sign"`                     // OpenList 签名
	Status              MediaStatus        `gorm:"index" json:"status"`                      // 状态
	SubtitleFiles       []*MediaMetaFiles  `json:"subtitle_files" gorm:"-"`                  // 整理后的字幕文件列表
	SubtitleFileJson    string             `json:"-"`                                        // SubtitleFiles 的 JSON 字符串
}

// 刮削好数据的集
type MediaEpisode struct {
	BaseModel
	MediaId           uint              `gorm:"index" json:"media_id"`        // 媒体 ID
	MediaSeasonId     uint              `gorm:"index" json:"media_season_id"` // 季 ID
	EpisodeName       string            `json:"episode_name"`                 // 集名称
	Overview          string            `json:"overview"`                     // 集描述
	PosterPath        string            `json:"poster_path"`                  // 海报路径 // TMDB 链接
	SeasonNumber      int               `gorm:"index" json:"season_number"`   // 季编号，例如：S01 中的 1
	EpisodeNumber     int               `gorm:"index" json:"episode_number"`  // 集编号，例如：S01E01 中的 1
	ReleaseDate       string            `json:"release_date"`                 // 发布日期
	VoteAverage       float64           `json:"vote_average"`                 // 投票平均分
	VoteCount         int64             `json:"vote_count"`                   // 投票数
	Actors            []helpers.Actor   `json:"actors" gorm:"-"`              // 演员列表
	ActorsJson        string            `json:"-"`                            // 演员列表 JSON 字符串
	Year              int               `json:"year"`                         // 年份
	VideoFileName     string            `json:"video_file_name"`              // 刮削整理后的电影或者电视剧的视频文件名
	VideoFileId       string            `json:"video_file_id"`                // 刮削整理后的电影或者电视剧的视频文件 ID，完整路径或者网盘文件 ID
	VideoPickCode     string            `json:"video_pick_code"`              // 115 PickCode 或百度网盘 fs_id
	VideoOpenListSign string            `json:"video_open_list_sign"`         // OpenList 签名
	Status            MediaStatus       `gorm:"index" json:"status"`          // 状态
	SubtitleFiles     []*MediaMetaFiles `json:"subtitle_files" gorm:"-"`      // 整理后的字幕文件列表
	SubtitleFileJson  string            `json:"-"`                            // SubtitleFiles 的 JSON 字符串
}

// 刮削好数据的季
type MediaSeason struct {
	BaseModel
	MediaId          uint        `gorm:"index" json:"media_id"`      // 媒体 ID
	SeasonNumber     int         `gorm:"index" json:"season_number"` // 季编号，例如：S01 中的 S1
	SeasonName       string      `json:"season_name"`                // 季名称，可能为空
	Overview         string      `json:"overview"`                   // 季描述
	NumberOfEpisodes int         `json:"number_of_episodes"`         // 集总数
	ReleaseDate      string      `json:"release_date"`               // 发布日期
	Path             string      `json:"path"`                       // 刮削整理后的季文件夹路径，固定值：Season + SeasonNumber
	PathId           string      `json:"path_id"`                    // 刮削整理后的季文件夹路径 ID，完整路径或者网盘文件 ID
	PosterPath       string      `json:"poster_path"`                // 海报路径 // TMDB 链接
	VoteAverage      float64     `json:"vote_average"`               // 投票平均分
	Year             int         `json:"year"`                       // 年份
	Status           MediaStatus `gorm:"index" json:"status"`        // 状态
}

func (m *Media) Save() error {
	// 将所有字段转换为 JSON 字符串
	m.ActorsJson = helpers.JsonString(m.Actors)
	m.DirectorJson = helpers.JsonString(m.Director)
	m.OriginalCountryJson = helpers.JsonString(m.OriginCountry)
	m.SubtitleFileJson = helpers.JsonString(m.SubtitleFiles)
	// 保存到数据库
	err := db.Db.Save(m).Error
	if err != nil {
		helpers.AppLogger.Errorf("保存 Media 到数据库失败：%v", err)
		return err
	}
	return nil
}

func (m *Media) DecodeJson() {
	// 解码 JSON 字符串
	if err := json.Unmarshal([]byte(m.ActorsJson), &m.Actors); err != nil {
		helpers.AppLogger.Warnf("解码 ActorsJson 失败：%v", err)
		m.Actors = []helpers.Actor{}
	}
	if err := json.Unmarshal([]byte(m.DirectorJson), &m.Director); err != nil {
		helpers.AppLogger.Warnf("解码 DirectorJson 失败：%v", err)
		m.Director = []helpers.Director{}
	}
	if err := json.Unmarshal([]byte(m.OriginalCountryJson), &m.OriginCountry); err != nil {
		helpers.AppLogger.Warnf("解码 OriginalCountryJson 失败：%v", err)
		m.OriginCountry = []string{}
	}
	if err := json.Unmarshal([]byte(m.SubtitleFileJson), &m.SubtitleFiles); err != nil {
		helpers.AppLogger.Warnf("解码 SubtitleFileJson 失败：%v", err)
		m.SubtitleFiles = []*MediaMetaFiles{}
	}
}

func (m *Media) UpdateSeasonCount(i int) {
	m.NumberOfSeasons += i
	// 保存
	updateData := map[string]interface{}{
		"number_of_seasons": m.NumberOfSeasons,
	}
	// 保存到数据库
	err := db.Db.Model(&Media{}).Where("id = ?", m.ID).Updates(updateData).Error
	if err != nil {
		helpers.AppLogger.Errorf("更新电视剧的总季数失败：%v", err)
	}
}

func (m *Media) UpdateEpisodeCount(i int) {
	m.NumberOfEpisodes += i
	// 保存
	updateData := map[string]interface{}{
		"number_of_episodes": m.NumberOfEpisodes,
	}
	// 保存到数据库
	err := db.Db.Model(&Media{}).Where("id = ?", m.ID).Updates(updateData).Error
	if err != nil {
		helpers.AppLogger.Errorf("更新电视剧的总剧集数失败：%v", err)
	}
}

func (ms *MediaSeason) Save() {
	// 保存到数据库
	err := db.Db.Save(ms).Error
	if err != nil {
		helpers.AppLogger.Errorf("保存季到数据库失败：%v", err)
	}
}

func (me *MediaEpisode) Save() {
	me.ActorsJson = helpers.JsonString(me.Actors)
	// 保存到数据库
	err := db.Db.Save(me).Error
	if err != nil {
		helpers.AppLogger.Errorf("保存剧集到数据库失败：%v", err)
	}
}

func (me *MediaEpisode) DecodeJson() {
	// 解码 JSON 字符串
	if err := json.Unmarshal([]byte(me.ActorsJson), &me.Actors); err != nil {
		helpers.AppLogger.Errorf("解码 ActorsJson 失败：%v", err)
	}
}

func GetMediaById(id uint) (*Media, error) {
	var media Media
	if err := db.Db.Where("id = ?", id).First(&media).Error; err != nil {
		return nil, err
	}
	// 解码 JSON 字符串
	media.DecodeJson()
	return &media, nil
}

func GetMediaSeasonById(id uint) (*MediaSeason, error) {
	var mediaSeason MediaSeason
	if err := db.Db.Where("id = ?", id).First(&mediaSeason).Error; err != nil {
		return nil, err
	}
	return &mediaSeason, nil
}

func GetMediaEpisodeById(id uint) (*MediaEpisode, error) {
	var mediaEpisode MediaEpisode
	if err := db.Db.Where("id = ?", id).First(&mediaEpisode).Error; err != nil {
		return nil, err
	}
	mediaEpisode.DecodeJson()
	return &mediaEpisode, nil
}

func GetMediaByTmdbId(tmdbId int64) (*Media, error) {
	var media Media
	if err := db.Db.Where("tmdb_id = ?", tmdbId).First(&media).Error; err != nil {
		return nil, err
	}
	// 解码 JSON 字符串
	media.DecodeJson()
	return &media, nil
}

// 如果 year = 0，则只用 name 查询
func GetMediaByName(mediaType MediaType, name string, year int) (*Media, error) {
	var media Media
	name = helpers.TitleCase(name)
	if year == 0 {
		if err := db.Db.Where("(name = ? OR original_name = ?) AND media_type = ?", name, name, mediaType).First(&media).Error; err != nil {
			return nil, err
		}
	} else {
		if err := db.Db.Where("(name = ? OR original_name = ?) AND year = ? AND media_type = ?", name, name, year, mediaType).First(&media).Error; err != nil {
			return nil, err
		}
	}
	// 解码 JSON 字符串
	media.DecodeJson()
	return &media, nil
}

func MakeMovieMediaFromNfo(movie *helpers.Movie) (*Media, error) {
	var media Media
	media.MediaType = MediaTypeMovie
	media.Name = movie.Title
	// 从 NFO 中提取信息
	if movie.Title != "" {
		media.Name = movie.Title
	}
	if movie.Year != 0 {
		media.Year = movie.Year
	}
	// 获取 TMDB ID
	for _, uid := range movie.Uniqueid {
		if uid.Type == "tmdb" {
			media.TmdbId = helpers.StringToInt64(uid.Id)
		}
		// 获取 IMDb ID
		if uid.Type == "imdb" {
			media.ImdbId = uid.Id
		}
	}
	if movie.Num != "" {
		media.Num = movie.Num
	}
	media.Runtime = movie.Runtime
	media.Num = movie.Num
	// 提取演员
	if movie.Actor != nil {
		media.Actors = movie.Actor
	}
	// 提取导演
	if movie.Director != nil {
		media.Director = movie.Director
	}
	if movie.MPAA != "" {
		media.MpaaRating = movie.MPAA
	}
	media.ReleaseDate = movie.Premiered
	media.Status = MediaStatusUnScraped
	return &media, nil
}

func GetSeasonByMediaIdAndSeasonNumber(mediaId uint, seasonNumber int) *MediaSeason {
	var season MediaSeason
	if err := db.Db.Where("media_id = ? AND season_number = ?", mediaId, seasonNumber).First(&season).Error; err != nil {
		return nil
	}
	return &season
}

func GetEpisodeByMediaIdAndSeasonNumber(mediaId uint, seasonNumber int, episodeNumber int) *MediaEpisode {
	var episode MediaEpisode
	err := db.Db.Where("media_id = ? AND season_number = ? AND episode_number = ?", mediaId, seasonNumber, episodeNumber).First(&episode).Error
	if err != nil {
		return nil
	}
	// 解码 JSON 字符串
	episode.DecodeJson()
	return &episode
}

func SetUnScrappedByMediaId(mediaId uint) {
	if mediaId == 0 {
		return
	}
	// 更新 MediaSeason 中对应 media_id 的状态为未刮削
	if err := db.Db.Model(&MediaSeason{}).Where("media_id = ?", mediaId).Update("status", MediaStatusUnScraped).Error; err != nil {
		return
	}
	// 更新 MediaEpisode 中对应 media_id 的状态为未刮削
	if err := db.Db.Model(&MediaEpisode{}).Where("media_id = ?", mediaId).Update("status", MediaStatusUnScraped).Error; err != nil {
		return
	}
}
