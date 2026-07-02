package synccron

import "testing"

func TestSyncRecordRetentionDays(t *testing.T) {
	if syncRecordRetentionDays != 7 {
		t.Fatalf("syncRecordRetentionDays = %d, want 7", syncRecordRetentionDays)
	}
}
