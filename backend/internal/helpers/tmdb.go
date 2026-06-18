package helpers

type Genre struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

var MovieGenres = []Genre{
	{Id: 28, Name: "动作"},
	{Id: 12, Name: "冒险"},
	{Id: 16, Name: "动画"},
	{Id: 35, Name: "喜剧"},
	{Id: 80, Name: "犯罪"},
	{Id: 99, Name: "纪录"},
	{Id: 18, Name: "剧情"},
	{Id: 10751, Name: "家庭"},
	{Id: 14, Name: "奇幻"},
	{Id: 36, Name: "历史"},
	{Id: 27, Name: "恐怖"},
	{Id: 10402, Name: "音乐"},
	{Id: 9648, Name: "悬疑"},
	{Id: 10749, Name: "爱情"},
	{Id: 878, Name: "科幻"},
	{Id: 10770, Name: "电视电影"},
	{Id: 53, Name: "惊悚"},
	{Id: 10752, Name: "战争"},
	{Id: 37, Name: "西部"},
}

var TvshowGenres = []Genre{
	{Id: 10759, Name: "动作与冒险"},
	{Id: 16, Name: "动画"},
	{Id: 35, Name: "喜剧"},
	{Id: 80, Name: "犯罪"},
	{Id: 99, Name: "纪录片"},
	{Id: 18, Name: "剧情"},
	{Id: 10751, Name: "家庭"},
	{Id: 10762, Name: "儿童"},
	{Id: 9648, Name: "悬疑"},
	{Id: 10763, Name: "新闻"},
	{Id: 10764, Name: "真人秀"},
	{Id: 10765, Name: "科幻与奇幻"},
	{Id: 10766, Name: "肥皂剧"},
	{Id: 10767, Name: "脱口秀"},
	{Id: 10768, Name: "战争与政治"},
	{Id: 37, Name: "西部片"},
}

// 默认语言
const DEFAULT_TMDB_LANGUAGE = "zh-CN"

// 备用语言
const BACKUP_TMDB_LANGUAGE = "en-US"

// 默认TMDB 图片语言
const DEFAULT_TMDB_IMAGE_LANGUAGE = "en-US"

// 默认TMDB API URL
const DEFAULT_TMDB_API_URL = "https://api.tmdb.org"

// 默认TMDB 图片URL
const DEFAULT_TMDB_IMAGE_URL = "https://image.tmdb.org"
