package migrate

import (
	"strings"

	"qmediasync/internal/validation"
)

type testDBRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

func (r *testDBRequest) Validate() error {
	r.trim()
	if err := validation.NonBlank("host", r.Host); err != nil {
		return err
	}
	if err := validation.RangeInt("port", r.Port, 1, 65535); err != nil {
		return err
	}
	return validation.NonBlank("user", r.User)
}

func (r *testDBRequest) trim() {
	r.Host = strings.TrimSpace(r.Host)
	r.User = strings.TrimSpace(r.User)
	r.Database = strings.TrimSpace(r.Database)
}

type saveConfigRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

func (r *saveConfigRequest) Validate() error {
	r.trim()
	if err := validation.NonBlank("host", r.Host); err != nil {
		return err
	}
	if err := validation.RangeInt("port", r.Port, 1, 65535); err != nil {
		return err
	}
	if err := validation.NonBlank("user", r.User); err != nil {
		return err
	}
	return validation.NonBlank("database", r.Database)
}

func (r *saveConfigRequest) trim() {
	r.Host = strings.TrimSpace(r.Host)
	r.User = strings.TrimSpace(r.User)
	r.Database = strings.TrimSpace(r.Database)
}
