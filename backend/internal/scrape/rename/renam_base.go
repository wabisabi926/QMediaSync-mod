package rename

import (
	"context"

	"qmediasync/internal/models"
)

type RenameBase struct {
	scrapePath *models.ScrapePath
	ctx        context.Context
}
