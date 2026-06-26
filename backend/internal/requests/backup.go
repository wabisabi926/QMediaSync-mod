package requests

import "qmediasync/internal/validation"

// BackupConfigUpdateRequest 更新备份配置请求。
type BackupConfigUpdateRequest struct {
	BackupEnabled   int    `json:"backup_enabled"`
	BackupCron      string `json:"backup_cron"`
	BackupRetention int    `json:"backup_retention"`
	BackupMaxCount  int    `json:"backup_max_count"`
	BackupCompress  int    `json:"backup_compress"`
}

// Validate 校验备份配置请求。
func (r BackupConfigUpdateRequest) Validate() error {
	if err := validation.OneOfInt("backup_enabled", r.BackupEnabled, []int{0, 1}); err != nil {
		return err
	}
	if err := validation.Cron("backup_cron", r.BackupCron, true); err != nil {
		return err
	}
	if r.BackupRetention > 0 {
		if err := validation.RangeInt("backup_retention", r.BackupRetention, 1, 365); err != nil {
			return err
		}
	}
	if err := validation.RangeInt("backup_max_count", r.BackupMaxCount, 0, 1000); err != nil {
		return err
	}
	return validation.OneOfInt("backup_compress", r.BackupCompress, []int{0, 1})
}
