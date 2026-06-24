package scrape

import (
	"context"

	"Q115-STRM/internal/models"
	"Q115-STRM/internal/tmdb"
)

// 从 TMDB 刮削元数据
type TmdbBase struct {
	scrapePath *models.ScrapePath
	ctx        context.Context
	Client     *tmdb.Client
}
