package scrape

import (
	"context"

	"qmediasync/internal/models"
)

type IdBase struct {
	tmdbImpl   TmdbImpl
	scrapePath *models.ScrapePath
	ctx        context.Context
}
