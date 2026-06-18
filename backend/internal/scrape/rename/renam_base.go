package rename

import (
	"Q115-STRM/internal/models"
	"context"
)

type RenameBase struct {
	scrapePath *models.ScrapePath
	ctx        context.Context
}
