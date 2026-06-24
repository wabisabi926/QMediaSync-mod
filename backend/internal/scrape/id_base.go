package scrape

import (
	"context"

	"Q115-STRM/internal/models"
)

type IdBase struct {
	tmdbImpl   TmdbImpl
	scrapePath *models.ScrapePath
	ctx        context.Context
}
