package controllers

import "testing"

func TestBuildNetFileListResponse(t *testing.T) {
	tests := []struct {
		name         string
		items        []*FileItem
		total        int64
		page         int
		pageSize     int
		wantTotal    int64
		wantPage     int
		wantPageSize int
	}{
		{
			name: "保留服务端返回的目录总数",
			items: []*FileItem{
				{Id: "1", Name: "电影.mkv", IsDirectory: false, Size: 1024, ModifiedAt: 100},
			},
			total:        305,
			page:         2,
			pageSize:     100,
			wantTotal:    305,
			wantPage:     2,
			wantPageSize: 100,
		},
		{
			name: "服务端没有总数时使用已加载条数兜底",
			items: []*FileItem{
				{Id: "1", Name: "a.mkv"},
				{Id: "2", Name: "b.mkv"},
			},
			total:        0,
			page:         1,
			pageSize:     100,
			wantTotal:    2,
			wantPage:     1,
			wantPageSize: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := buildNetFileListResponse(tt.items, tt.total, tt.page, tt.pageSize)

			if len(response.List) != len(tt.items) {
				t.Fatalf("list 数量 = %d，期望 %d", len(response.List), len(tt.items))
			}
			if response.Total != tt.wantTotal {
				t.Fatalf("total = %d，期望 %d", response.Total, tt.wantTotal)
			}
			if response.Page != tt.wantPage {
				t.Fatalf("page = %d，期望 %d", response.Page, tt.wantPage)
			}
			if response.PageSize != tt.wantPageSize {
				t.Fatalf("page_size = %d，期望 %d", response.PageSize, tt.wantPageSize)
			}
		})
	}
}
