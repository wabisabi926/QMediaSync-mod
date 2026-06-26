package requests

import (
	"testing"

	"qmediasync/internal/models"
)

func TestPaginationRequestValidate(t *testing.T) {
	req := PaginationRequest{}
	if err := req.Normalize(20); err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if req.Page != 1 || req.PageSize != 20 {
		t.Fatalf("Normalize() = %+v", req)
	}

	req = PaginationRequest{Page: 1, PageSize: 101}
	if err := req.Normalize(20); err == nil {
		t.Fatal("Normalize() error = nil, want error")
	}

	req = PaginationRequest{Page: 1, PageSize: 1151}
	if err := req.NormalizeFileList(); err == nil {
		t.Fatal("NormalizeFileList() error = nil, want error")
	}
}

func TestOperationRequestValidate(t *testing.T) {
	t.Run("正 ID 通过", func(t *testing.T) {
		req := PositiveIDRequest{ID: 1}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("ID 列表去重通过", func(t *testing.T) {
		req := IDListRequest{IDs: []uint{2, 1, 2}}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if got := req.NormalizedIDs(); len(got) != 2 || got[0] != 2 || got[1] != 1 {
			t.Fatalf("NormalizedIDs() = %+v", got)
		}
	})

	t.Run("ID 列表包含 0 失败", func(t *testing.T) {
		req := IDListRequest{IDs: []uint{1, 0}}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})
}

func TestAssociationRequestValidate(t *testing.T) {
	t.Run("同步路径关联允许清空刮削路径", func(t *testing.T) {
		req := SaveRelScrapePathRequest{SyncPathID: 1, ScrapePathIDs: []uint{}}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("刮削路径关联允许清空同步路径", func(t *testing.T) {
		req := SaveScrapeStrmPathRequest{ScrapePathID: 1, SyncPathIDs: []uint{}}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("关联列表包含 0 失败", func(t *testing.T) {
		req := SaveRelScrapePathRequest{SyncPathID: 1, ScrapePathIDs: []uint{1, 0}}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})
}

func TestIDCSVRequestValidate(t *testing.T) {
	t.Run("逗号 ID 列表通过并去重", func(t *testing.T) {
		req := IDCSVRequest{IDs: "2,1,2"}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if got := req.NormalizedIDs(); len(got) != 2 || got[0] != 2 || got[1] != 1 {
			t.Fatalf("NormalizedIDs() = %+v", got)
		}
	})

	t.Run("逗号 ID 列表包含 0 失败", func(t *testing.T) {
		req := IDCSVRequest{IDs: "1,0"}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})
}

func TestPathRequestValidate(t *testing.T) {
	t.Run("创建本地目录通过", func(t *testing.T) {
		req := CreateDirRequest{SourceType: models.SourceTypeLocal, ParentID: "/tmp", Name: "movies"}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("远程创建目录缺少账号失败", func(t *testing.T) {
		req := CreateDirRequest{SourceType: models.SourceType115, ParentID: "0", Name: "movies"}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})

	t.Run("文件夹名路径穿越失败", func(t *testing.T) {
		req := CreateDirRequest{SourceType: models.SourceTypeLocal, ParentID: "/tmp", Name: ".."}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})
}

func TestQueueRequestValidate(t *testing.T) {
	req := QueueListRequest{PaginationRequest: PaginationRequest{Page: 1, PageSize: 100}}
	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestLogRequestValidate(t *testing.T) {
	t.Run("旧日志请求通过", func(t *testing.T) {
		req := OldLogsRequest{Path: "app.log", Limit: 100, Direction: "forward"}
		if err := req.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("日志路径穿越失败", func(t *testing.T) {
		req := LogFileRequest{Path: "../app.log"}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want error")
		}
	})
}

func TestUpdateRequestValidate(t *testing.T) {
	req := UpdateVersionRequest{Version: "v1.2.3"}
	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	req = UpdateVersionRequest{Version: "latest"}
	if err := req.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}

func TestTmpImageRequestValidate(t *testing.T) {
	req := TmpImageRequest{Path: "poster.jpg", MediaType: models.MediaTypeMovie}
	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	req = TmpImageRequest{Path: "/etc/passwd", MediaType: models.MediaTypeMovie}
	if err := req.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}

func TestQueueStatsRequestValidate(t *testing.T) {
	req := QueueStatsRequest{TimeWindow: 3600, StartDate: "2026-06-01", EndDate: "2026-06-15"}
	if err := req.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	req = QueueStatsRequest{TimeWindow: 59}
	if err := req.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}
