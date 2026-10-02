package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"qmediasync/internal/helpers"
	"qmediasync/internal/models"
	"qmediasync/internal/requests"
	"qmediasync/internal/v115open"

	"github.com/gin-gonic/gin"
)

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
			response := buildNetFileListResponse(netFileListResponseOptions{
				List:     tt.items,
				Total:    tt.total,
				Page:     tt.page,
				PageSize: tt.pageSize,
			})

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

func TestSplitOpenListFileIDs(t *testing.T) {
	dir, names, err := splitOpenListFileIDs("/Movies", []string{"/Movies/A.mkv", "/Movies/B"})
	if err != nil {
		t.Fatalf("splitOpenListFileIDs() error = %v", err)
	}
	if dir != "/Movies" || len(names) != 2 || names[0] != "A.mkv" || names[1] != "B" {
		t.Fatalf("dir = %s, names = %+v", dir, names)
	}

	dir, names, err = splitOpenListFileIDs("", []string{"/Movies/A.mkv", "/Movies/B.mkv"})
	if err != nil {
		t.Fatalf("splitOpenListFileIDs() error = %v", err)
	}
	if dir != "/Movies" || len(names) != 2 || names[0] != "A.mkv" || names[1] != "B.mkv" {
		t.Fatalf("dir = %s, names = %+v", dir, names)
	}

	if _, _, err := splitOpenListFileIDs("/", []string{"/"}); err == nil {
		t.Fatal("splitOpenListFileIDs() error = nil, want error for root path")
	}
	if _, _, err := splitOpenListFileIDs("", []string{"/Movies/A.mkv", "/Other/B.mkv"}); err == nil {
		t.Fatal("跨目录批量操作必须在提交前被拒绝")
	}
}

func setupOpenListFileOperationTest(t *testing.T, handler http.HandlerFunc) *models.Account {
	t.Helper()
	db := setupControllerTestDB(t, &models.Account{})
	previousLogger, previousCache := helpers.OpenListLog, netFileCache
	helpers.OpenListLog = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	netFileCache = newNetFileBatchCache(200, time.Minute)
	t.Cleanup(func() { helpers.OpenListLog, netFileCache = previousLogger, previousCache })
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	account := &models.Account{Name: t.Name(), SourceType: models.SourceTypeOpenList, BaseUrl: server.URL, Token: "fixture-token"}
	if err := db.Create(account).Error; err != nil {
		t.Fatal(err)
	}
	return account
}

func TestRead115DirectoryPages(t *testing.T) {
	for _, tc := range []struct {
		name                                      string
		total, directoryCount, pageSize, sysCount int
		first                                     bool
		filesAtStart                              bool
		wantCalls                                 int
	}{
		{name: "591目录不截断200", total: 591, directoryCount: 591, pageSize: 200, wantCalls: 3},
		{name: "跨1000分页", total: 1591, directoryCount: 1591, pageSize: 1000, wantCalls: 2},
		{name: "系统目录已计入count", total: 7, directoryCount: 7, pageSize: 2, sysCount: 2, first: true, wantCalls: 4},
		{name: "置顶时不枚举剩余文件", total: 3000, directoryCount: 591, pageSize: 1000, first: true, wantCalls: 1},
		{name: "跟随模式越过文件继续找目录", total: 1591, directoryCount: 591, pageSize: 1000, filesAtStart: true, wantCalls: 2},
		{name: "空目录", pageSize: 1000, wantCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, nextOffset := 0, 0
			got, err := read115DirectoryPages(t.Context(), tc.first, func(offset int) (*v115open.FileListResp, error) {
				calls++
				if offset != nextOffset {
					t.Fatalf("offset=%d，期望原始条目起点%d", offset, nextOffset)
				}
				resp := &v115open.FileListResp{Count: tc.total, SysCount: tc.sysCount, PathStr: "parent"}
				resp.State = true
				for i := offset; i < min(offset+tc.pageSize, tc.total); i++ {
					category := v115open.TypeFile
					if (!tc.filesAtStart && i < tc.directoryCount) || (tc.filesAtStart && i >= tc.total-tc.directoryCount) {
						category = v115open.TypeDir
					}
					resp.Data = append(resp.Data, v115open.File{FileId: strconv.Itoa(i + 1), FileName: fmt.Sprintf("item-%04d", tc.total-i), FileCategory: category})
				}
				nextOffset += len(resp.Data)
				return resp, nil
			})
			if err != nil || len(got) != tc.directoryCount || calls != tc.wantCalls {
				t.Fatalf("目录数=%d，请求数=%d，错误=%v", len(got), calls, err)
			}
			if len(got) > 1 && got[0].Name <= got[1].Name {
				t.Fatal("不能自行按名称重排远端目录")
			}
		})
	}
}

func TestRead115DirectoryPagesRejectsIncompleteResults(t *testing.T) {
	for _, tc := range []struct{ name, mode string }{
		{name: "上游失败", mode: "error"}, {name: "意外空页", mode: "empty"},
		{name: "重复分页", mode: "repeat"}, {name: "取消后不交付部分结果", mode: "cancel"},
		{name: "失败标记", mode: "state"}, {name: "空响应", mode: "nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			calls := 0
			got, err := read115DirectoryPages(ctx, false, func(offset int) (*v115open.FileListResp, error) {
				calls++
				resp := &v115open.FileListResp{Count: 2}
				resp.State = true
				resp.Data = []v115open.File{{FileId: "1", FileName: "first", FileCategory: v115open.TypeDir}}
				if calls == 1 {
					return resp, nil
				}
				switch tc.mode {
				case "error":
					return nil, errors.New("upstream unavailable")
				case "empty":
					resp.Data = nil
				case "repeat":
				case "cancel":
					cancel()
					resp.Data[0].FileId = "2"
				case "state":
					resp.State = false
				case "nil":
					return nil, nil
				}
				return resp, nil
			})
			if err == nil {
				t.Fatalf("不完整结果不能成功返回：%+v", got)
			}
			if calls != 2 {
				t.Fatalf("请求次数=%d，期望2", calls)
			}
		})
	}
}

func TestRead115DirectoryPagesRejectsUnknownCategory(t *testing.T) {
	for _, first := range []bool{false, true} {
		for _, category := range []v115open.FileType{"", "unknown"} {
			t.Run(fmt.Sprintf("foldersFirst=%t/category=%q", first, category), func(t *testing.T) {
				calls := 0
				got, err := read115DirectoryPages(t.Context(), first, func(offset int) (*v115open.FileListResp, error) {
					calls++
					resp := &v115open.FileListResp{Count: 3}
					resp.State = true
					resp.Data = []v115open.File{
						{FileId: "1", FileName: "directory", FileCategory: v115open.TypeDir},
						{FileId: "2", FileName: "unclassified", FileCategory: category},
					}
					return resp, nil
				})
				if err == nil || got != nil || calls != 1 {
					t.Fatalf("未知类别不能作为文件结束目录前缀：目录=%v，错误=%v，请求数=%d", got, err, calls)
				}
			})
		}
	}
}

func TestOpenListDirectoryBrowseRefreshAndOrder(t *testing.T) {
	for _, tc := range []struct {
		name             string
		refresh, failure bool
	}{
		{name: "直接原生目录列表"}, {name: "显式刷新后读取", refresh: true}, {name: "刷新失败停止", refresh: true, failure: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			paths := make([]string, 0)
			account := setupOpenListFileOperationTest(t, func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["path"] != "/parent " {
					t.Errorf("路径空白丢失：%v", body)
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/api/fs/list":
					if body["refresh"] != true || body["page"] != float64(1) || body["per_page"] != float64(1) {
						t.Errorf("刷新参数=%v", body)
					}
					if tc.failure {
						_, _ = io.WriteString(w, `{"code":500,"message":"refresh failed"}`)
						return
					}
					_, _ = io.WriteString(w, `{"code":200,"data":{"content":[],"total":200}}`)
				case "/api/fs/dirs":
					if body["force_root"] != false {
						t.Errorf("不允许force_root：%v", body)
					}
					if _, exists := body["refresh"]; exists {
						t.Error("dirs不支持refresh参数")
					}
					_, _ = io.WriteString(w, `{"code":200,"data":[{"name":"Z"},{"name":"A "}]}`)
				default:
					t.Errorf("意外路径：%s", r.URL.Path)
				}
			})
			got, err := GetOpenListPath(t.Context(), "/parent ", account, tc.refresh)
			wantPaths := []string{"/api/fs/dirs"}
			if tc.refresh {
				wantPaths = []string{"/api/fs/list", "/api/fs/dirs"}
			}
			if tc.failure {
				wantPaths = make([]string, len(paths))
				for i := range wantPaths {
					wantPaths[i] = "/api/fs/list"
				}
				if len(paths) == 0 {
					t.Fatal("应先尝试刷新")
				}
			}
			if !slices.Equal(paths, wantPaths) || (err != nil) != tc.failure {
				t.Fatalf("请求=%v，错误=%v", paths, err)
			}
			if !tc.failure && (len(got) != 2 || got[0].Name != "Z" || got[1].Path != "/parent /A ") {
				t.Fatalf("目录顺序或路径错误：%+v", got)
			}
		})
	}
}

func TestBaiduDirectoryBrowseUsesListPagination(t *testing.T) {
	previousCache := netFileCache
	netFileCache = newNetFileBatchCache(200, 180*time.Second)
	t.Cleanup(func() { netFileCache = previousCache })
	setupControllerTestDB(t, &models.Account{})
	previousTransport, previousLogger := http.DefaultTransport, helpers.BaiduPanLog
	helpers.BaiduPanLog = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	t.Cleanup(func() { http.DefaultTransport, helpers.BaiduPanLog = previousTransport, previousLogger })
	calls := 0
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		query := r.URL.Query()
		wantStart := strconv.Itoa(calls * 1000)
		calls++
		if r.URL.Path != "/rest/2.0/xpan/file" || query.Get("method") != "list" || query.Get("folder") != "1" || query.Get("start") != wantStart || query.Get("limit") != "1000" || query.Get("order") != "time" || query.Get("desc") != "1" || query.Get("web") != "1" {
			t.Fatalf("百度列表请求错误：%s %v", r.URL.Path, query)
		}
		items := make([]map[string]string, 0)
		start, _ := strconv.Atoi(wantStart)
		for i := start; i < min(start+1000, 1591); i++ {
			items = append(items, map[string]string{"path": fmt.Sprintf("/parent /dir-%04d ", 1591-i)})
		}
		payload, _ := json.Marshal(map[string]any{"errno": 0, "list": items})
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(payload)), Request: r}, nil
	})
	account := &models.Account{ID: 900001, SourceType: models.SourceTypeBaiduPan, Token: "fixture-token"}
	got, err := GetBaiduPanPathList(t.Context(), requests.PathListRequest{ParentID: "/parent ", SortBy: "time", SortOrder: "desc"}, account)
	if err != nil || len(got) != 1591 || calls != 2 {
		t.Fatalf("目录数=%d，请求数=%d，错误=%v", len(got), calls, err)
	}
	if got[0].Id != "parent /dir-1591 " || got[1590].Id != "parent /dir-0001 " {
		t.Fatal("百度路径空白或原始顺序被修改")
	}
}

func TestLocalDirectoryBrowseIncludesModificationTime(t *testing.T) {
	parent := t.TempDir()
	directory := filepath.Join(parent, "visible")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(parent, ".hidden"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "file.txt"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	modified := time.Unix(1234567890, 0)
	if err := os.Chtimes(directory, modified, modified); err != nil {
		t.Fatal(err)
	}
	got, err := GetLocalPath(parent)
	if err != nil || len(got) != 1 || got[0].Name != "visible" || got[0].ModifiedTime == nil || *got[0].ModifiedTime != modified.Unix() {
		t.Fatalf("目录列表=%+v，错误=%v", got, err)
	}
}

func TestCreateLocalDirectoryIDMatchesBrowse(t *testing.T) {
	parent := filepath.ToSlash(t.TempDir())
	body, err := json.Marshal(requests.CreateDirRequest{
		ParentID: parent, ParentPath: parent, SourceType: models.SourceTypeLocal, Name: "new folder",
	})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/path/create", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	CreateDir(c)
	var result APIResponse[DirResp]
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || result.Code != Success {
		t.Fatalf("创建目录失败：%s", w.Body.String())
	}
	directories, err := GetLocalPath(parent)
	if err != nil || len(directories) != 1 {
		t.Fatalf("目录列表=%+v，错误=%v", directories, err)
	}
	want := parent + "/new folder"
	if result.Data.Id != want || result.Data.Path != want || directories[0].Id != result.Data.Id {
		t.Fatalf("创建结果=%+v，刷新结果=%+v，期望路径=%q", result.Data, directories[0], want)
	}
}

func TestGetPathListRejectsAccountSourceMismatch(t *testing.T) {
	calls := 0
	account := setupOpenListFileOperationTest(t, func(w http.ResponseWriter, r *http.Request) { calls++ })
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/path/list?source_type=%s&account_id=%d", models.SourceType115, account.ID), nil)
	GetPathList(c)
	var result APIResponse[any]
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Code == Success || calls != 0 || !strings.Contains(result.Message, "账号与来源类型不匹配") {
		t.Fatalf("来源不匹配响应=%s，上游请求=%d", w.Body.String(), calls)
	}
}

func TestGetBrowseSortOptionsResponse(t *testing.T) {
	for _, tc := range []struct {
		name, query string
		success     bool
	}{
		{name: "115文件能力", query: "source_type=115&scope=files", success: true},
		{name: "来源非法", query: "source_type=unknown&scope=files"},
		{name: "场景缺失", query: "source_type=115"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/path/sort-options?"+tc.query, nil)
			GetBrowseSortOptions(c)
			var result APIResponse[requests.BrowseSortOptions]
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if w.Code != http.StatusOK || (result.Code == Success) != tc.success {
				t.Fatalf("能力响应=%s", w.Body.String())
			}
			if tc.success && (!result.Data.FoldersFirst || result.Data.Default.FoldersFirst == nil || !*result.Data.Default.FoldersFirst || result.Data.Default.SortBy != "name" || !slices.Contains(result.Data.Fields, "default")) {
				t.Fatalf("115默认能力不完整：%s", w.Body.String())
			}
		})
	}
}

func Test115BrowseSharesRawBatchAcrossFilesAndDirectories(t *testing.T) {
	previous := netFileCache
	netFileCache = newNetFileBatchCache(200, 180*time.Second)
	t.Cleanup(func() { netFileCache = previous })
	account := &models.Account{ID: 900011, SourceType: models.SourceType115}
	raw := &v115open.FileListResp{Count: 3, PathStr: "parent"}
	raw.State = true
	raw.Data = []v115open.File{
		{FileId: "1", FileName: "Z folder", FileCategory: v115open.TypeDir},
		{FileId: "2", FileName: "A folder", FileCategory: v115open.TypeDir},
		{FileId: "3", FileName: "video.mp4", FileCategory: v115open.TypeFile, FileSize: 42, Utime: 1234},
	}
	key := netFileBatchCacheKey{
		SourceType: "115", AccountID: account.ID, Path: "20", SortBy: "name", SortOrder: "asc",
		FoldersFirst: true, Filter: "none", BatchSize: 1000,
	}
	calls := 0
	_, _, err := netFileCache.getOrFetch(t.Context(), key, false, func(context.Context) (netFileBatch, error) {
		calls++
		return netFileBatch{Raw115: raw, Total: 3, TotalExact: true}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// 空 Token 确保任一入口若错误地绕过共享批次就会失败，测试不会访问真实网盘。
	for range 2 {
		dirs, err := Get115PathList(t.Context(), requests.PathListRequest{ParentID: "20", SortBy: "name", SortOrder: "asc"}, account)
		if err != nil || len(dirs) != 2 || dirs[0].Path != "parent/Z folder" || dirs[1].Id != "2" {
			t.Fatalf("目录过滤必须共享并保序：%+v %v", dirs, err)
		}
		files, err := getNetFileListPage(t.Context(), netFileListQuery{Account: account, ParentID: "20", SortBy: "name", SortOrder: "asc", Page: 1, PageSize: 50})
		if err != nil || len(files.List) != 3 || files.List[2].Size != 42 || files.List[2].ModifiedAt != 1234 || files.Cache.Status != netFileCacheHit {
			t.Fatalf("目录过滤不能覆盖文件批次：%+v %v", files, err)
		}
	}
	if batch, ok := netFileCache.Get(key, time.Now()); !ok || batch.Raw115 != raw || batch.Items != nil || calls != 1 {
		t.Fatal("115 应只保留同一份原始批次")
	}
	if _, err := Get115PathList(t.Context(), requests.PathListRequest{ParentID: "20"}, account); err == nil {
		t.Fatal("旧目录请求不能误用显式名称排序缓存")
	}
}

func TestOpenListBrowseSharesInvalidationButNotEndpointData(t *testing.T) {
	lists, dirs := 0, 0
	account := setupOpenListFileOperationTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/fs/list":
			lists++
			_, _ = io.WriteString(w, `{"code":200,"data":{"content":[{"name":"file.mkv","is_dir":false}],"total":1}}`)
		case "/api/fs/dirs":
			dirs++
			_, _ = io.WriteString(w, `{"code":200,"data":[{"name":"folder"}]}`)
		default:
			t.Errorf("意外请求 %s", r.URL.Path)
		}
	})
	readFiles := func(refresh bool) {
		t.Helper()
		files, err := getNetFileListPage(t.Context(), netFileListQuery{Account: account, ParentID: "/parent", Page: 1, PageSize: 50, Refresh: refresh})
		if err != nil || len(files.List) != 1 || files.List[0].Name != "file.mkv" {
			t.Fatalf("文件列表=%+v %v", files, err)
		}
	}
	readDirs := func(refresh bool) {
		t.Helper()
		folders, err := GetOpenListPath(t.Context(), "/parent", account, refresh)
		if err != nil || len(folders) != 1 || folders[0].Name != "folder" {
			t.Fatalf("目录列表=%+v %v", folders, err)
		}
	}
	for range 2 {
		readFiles(false)
		readDirs(false)
	}
	if lists != 1 || dirs != 1 {
		t.Fatalf("两个原生接口各请求一次后应缓存：list=%d dirs=%d", lists, dirs)
	}
	readDirs(true)
	readFiles(false)
	if lists != 3 || dirs != 2 {
		t.Fatalf("刷新目录应失效文件视图：list=%d dirs=%d", lists, dirs)
	}
	readFiles(true)
	readDirs(false)
	if lists != 4 || dirs != 3 {
		t.Fatalf("刷新文件应失效目录视图：list=%d dirs=%d", lists, dirs)
	}
}

func TestBaiduBrowseCacheSeparatesFolderFilterAndRefreshesAllSorts(t *testing.T) {
	setupControllerTestDB(t, &models.Account{})
	previousTransport, previousCache := http.DefaultTransport, netFileCache
	previousLogger := helpers.BaiduPanLog
	helpers.BaiduPanLog = &helpers.QLogger{Logger: log.New(io.Discard, "", 0)}
	netFileCache = newNetFileBatchCache(200, 180*time.Second)
	t.Cleanup(func() {
		http.DefaultTransport, netFileCache = previousTransport, previousCache
		helpers.BaiduPanLog = previousLogger
	})
	calls := make(map[string]int)
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		q := r.URL.Query()
		calls[q.Get("folder")+":"+q.Get("order")]++
		payload := `{"errno":0,"list":[{"path":"/parent/file.mkv","isdir":0}]}`
		if q.Get("folder") == "1" {
			payload = `{"errno":0,"list":[{"path":"/parent/folder"}]}`
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(payload)), Request: r}, nil
	})
	account := &models.Account{ID: 900012, SourceType: models.SourceTypeBaiduPan, Token: "fixture-token"}
	readFiles := func(sortBy string) {
		t.Helper()
		files, err := getNetFileListPage(t.Context(), netFileListQuery{Account: account, ParentID: "/parent", SortBy: sortBy, Page: 1, PageSize: 50})
		if err != nil || len(files.List) != 1 || files.List[0].Id != "/parent/file.mkv" {
			t.Fatalf("文件列表=%+v %v", files, err)
		}
	}
	readDirs := func(refresh int) {
		t.Helper()
		folders, err := GetBaiduPanPathList(t.Context(), requests.PathListRequest{ParentID: "/parent", SortBy: "name", SortOrder: "asc", Refresh: refresh}, account)
		if err != nil || len(folders) != 1 || folders[0].Id != "parent/folder" {
			t.Fatalf("目录列表=%+v %v", folders, err)
		}
	}
	for range 2 {
		readFiles("name")
		readFiles("time")
		readDirs(0)
	}
	for _, key := range []string{"0:name", "0:time", "1:name"} {
		if calls[key] != 1 {
			t.Fatalf("不同筛选及排序应独立缓存：%v", calls)
		}
	}
	readDirs(1)
	readFiles("name")
	readFiles("time")
	for _, key := range []string{"0:name", "0:time", "1:name"} {
		if calls[key] != 2 {
			t.Fatalf("刷新目录应失效全部排序及文件视图：%v", calls)
		}
	}
}

func TestValidate115BrowseBatchRejectsCountMismatch(t *testing.T) {
	for _, tc := range []struct {
		name         string
		start, count int
	}{
		{name: "非空页但总数为零"},
		{name: "后页超过总数", start: 10, count: 10},
		{name: "后页超出总数范围", start: 20, count: 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &v115open.FileListResp{Count: tc.count}
			resp.State = true
			resp.Data = []v115open.File{{FileId: "1", FileCategory: v115open.TypeDir}}
			if err := validate115BrowseBatch(resp, tc.start); err == nil {
				t.Fatal("不能缓存条目数量超过上游总数的响应")
			}
		})
	}
}
