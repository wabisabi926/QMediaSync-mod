package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUpdateProgressReturnsSnapshotForTerminalStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldInfo := currentUpdateInfo
	oldCancel := currentUpdateCancel
	t.Cleanup(func() {
		currentUpdateInfo = oldInfo
		currentUpdateCancel = oldCancel
	})

	setCurrentUpdateInfoForTest(&updateInfo{
		Version:    "v0.16.0",
		Progress:   100,
		TotalSize:  100,
		Downloaded: 100,
		Status:     string(updateStatusCompleted),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/update/progress", nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	UpdateProgress(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("HTTP 状态码 = %d，期望 %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !containsAll(body, []string{`"code":200`, `"status":"completed"`, `"progress":100`}) {
		t.Fatalf("响应体 = %s，期望包含 completed 终态快照", body)
	}
}

func TestUpdateProgressReturnsBadRequestWhenNoSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldInfo := currentUpdateInfo
	oldCancel := currentUpdateCancel
	t.Cleanup(func() {
		currentUpdateInfo = oldInfo
		currentUpdateCancel = oldCancel
	})

	setCurrentUpdateInfoForTest(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/update/progress", nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	UpdateProgress(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("HTTP 状态码 = %d，期望 %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); !containsAll(body, []string{`"code":500`, "未开始更新"}) {
		t.Fatalf("响应体 = %s，期望提示未开始更新", body)
	}
}

func containsAll(raw string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(raw, part) {
			return false
		}
	}
	return true
}
