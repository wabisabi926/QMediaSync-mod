package rename

import (
	"context"

	"Q115-STRM/internal/models"
)

type RenameBase struct {
	scrapePath *models.ScrapePath
	ctx        context.Context
}
