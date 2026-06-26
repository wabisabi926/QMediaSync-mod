package validation

import "fmt"

type Error struct {
	Field   string
	Message string
}

func (e Error) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s：%s", e.Field, e.Message)
}

func New(field string, message string) error {
	return Error{Field: field, Message: message}
}
