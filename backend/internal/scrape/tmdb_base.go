package scrape

import (
	"Q115-STRM/internal/models"
	"Q115-STRM/internal/tmdb"
	"context"
)

// 从tmdb刮削元数据
type TmdbBase struct {
	scrapePath *models.ScrapePath
	ctx        context.Context
	Client     *tmdb.Client
}
