package models

// SyncPathIdempotencyRecord 保存同步目录创建请求的幂等结果。
type SyncPathIdempotencyRecord struct {
	BaseModel
	KeyHash    string `json:"key_hash" gorm:"size:64;uniqueIndex"`
	SyncPathId uint   `json:"sync_path_id" gorm:"index"`
	Status     string `json:"status" gorm:"size:32;index"`
}

func (*SyncPathIdempotencyRecord) TableName() string {
	return "sync_path_idempotency_records"
}
