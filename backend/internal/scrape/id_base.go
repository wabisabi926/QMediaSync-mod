package scrape

import (
	"Q115-STRM/internal/models"
	"context"
)

type IdBase struct {
	tmdbImpl   TmdbImpl
	scrapePath *models.ScrapePath
	ctx        context.Context
}
