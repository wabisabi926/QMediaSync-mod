package embyclientrestgo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestFetchMediaItemsByLibraryID分页流式处理(t *testing.T) {
	requestedStartIndexes := []int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startIndex, _ := strconv.Atoi(r.URL.Query().Get("StartIndex"))
		requestedStartIndexes = append(requestedStartIndexes, startIndex)

		switch startIndex {
		case 0:
			fmt.Fprint(w, `{"TotalRecordCount":5,"Items":[{"Id":"1"},{"Id":"2"}]}`)
		case 2:
			fmt.Fprint(w, `{"TotalRecordCount":5,"Items":[{"Id":"3"},{"Id":"4"}]}`)
		case 4:
			fmt.Fprint(w, `{"TotalRecordCount":5,"Items":[{"Id":"5"}]}`)
		default:
			t.Fatalf("unexpected StartIndex %d", startIndex)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ids := []string{}
	err := client.FetchMediaItemsByLibraryID(
		context.Background(),
		EmbyItemsQuery{LibraryID: "lib-1", Limit: 2},
		func(item BaseItemDtoV2) error {
			ids = append(ids, item.Id)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("FetchMediaItemsByLibraryID() error = %v", err)
	}
	if got := fmt.Sprint(ids); got != "[1 2 3 4 5]" {
		t.Fatalf("handled ids = %s, want [1 2 3 4 5]", got)
	}
	if got := fmt.Sprint(requestedStartIndexes); got != "[0 2 4]" {
		t.Fatalf("requested StartIndex = %s, want [0 2 4]", got)
	}
}

func TestFetchMediaItemsByLibraryID空页停止(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		startIndex, _ := strconv.Atoi(r.URL.Query().Get("StartIndex"))
		if startIndex == 0 {
			fmt.Fprint(w, `{"TotalRecordCount":10,"Items":[{"Id":"1"}]}`)
			return
		}
		fmt.Fprint(w, `{"TotalRecordCount":10,"Items":[]}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	handled := 0
	err := client.FetchMediaItemsByLibraryID(
		context.Background(),
		EmbyItemsQuery{LibraryID: "lib-1", Limit: 1},
		func(item BaseItemDtoV2) error {
			handled++
			return nil
		},
	)
	if err != nil {
		t.Fatalf("FetchMediaItemsByLibraryID() error = %v", err)
	}
	if handled != 1 || requests != 2 {
		t.Fatalf("handled=%d requests=%d, want handled=1 requests=2", handled, requests)
	}
}

func TestFetchMediaItemsByLibraryIDHandle错误停止分页(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests > 1 {
			t.Fatal("handle 返回错误后不应继续请求下一页")
		}
		fmt.Fprint(w, `{"TotalRecordCount":3,"Items":[{"Id":"1"},{"Id":"2"}]}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	wantErr := errors.New("stop")
	err := client.FetchMediaItemsByLibraryID(
		context.Background(),
		EmbyItemsQuery{LibraryID: "lib-1", Limit: 2},
		func(item BaseItemDtoV2) error {
			if item.Id == "2" {
				return wantErr
			}
			return nil
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("FetchMediaItemsByLibraryID() error = %v, want %v", err, wantErr)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestFetchMediaItemsByLibraryIDLastDateCreatedAt正常停止(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests > 1 {
			t.Fatal("达到 LastDateCreatedAt 截止点后不应继续请求下一页")
		}
		fmt.Fprint(w, `{"TotalRecordCount":3,"Items":[{"Id":"1","DateCreated":"2026-06-29T01:00:00Z"},{"Id":"2","DateCreated":"2026-06-29T00:00:00Z"}]}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ids := []string{}
	err := client.FetchMediaItemsByLibraryID(
		context.Background(),
		EmbyItemsQuery{LibraryID: "lib-1", Limit: 2, LastDateCreatedAt: 1782691200},
		func(item BaseItemDtoV2) error {
			ids = append(ids, item.Id)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("FetchMediaItemsByLibraryID() error = %v, want nil", err)
	}
	if got := fmt.Sprint(ids); got != "[1]" {
		t.Fatalf("handled ids = %s, want [1]", got)
	}
}

func TestFetchMediaItemsByLibraryIDContext取消(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("context 已取消时不应发送请求")
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.FetchMediaItemsByLibraryID(
		ctx,
		EmbyItemsQuery{LibraryID: "lib-1"},
		func(item BaseItemDtoV2) error { return nil },
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("FetchMediaItemsByLibraryID() error = %v, want context.Canceled", err)
	}
}

func TestFetchMediaItemsByLibraryIDHTTP错误(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	err := client.FetchMediaItemsByLibraryID(
		context.Background(),
		EmbyItemsQuery{LibraryID: "lib-1"},
		func(item BaseItemDtoV2) error { return nil },
	)
	if err == nil {
		t.Fatal("FetchMediaItemsByLibraryID() error = nil, want error")
	}
}
