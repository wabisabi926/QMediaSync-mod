package v115open

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestParseSignCheckRange(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantStart int64
		wantEnd   int64
		wantErr   bool
	}{
		{name: "合法闭区间", value: "0-131071", wantStart: 0, wantEnd: 131071},
		{name: "非法格式", value: "0:131071", wantErr: true},
		{name: "结束位置小于起始位置", value: "100-99", wantErr: true},
		{name: "非数字", value: "a-99", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSignCheckRange(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("期望解析失败，实际返回 nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("解析 sign_check 失败：%v", err)
			}
			if got.Start != tt.wantStart || got.End != tt.wantEnd {
				t.Fatalf("range = %+v，期望 %d-%d", got, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestBuildOSSCallbackHeaders(t *testing.T) {
	headers, err := BuildOSSCallbackHeaders(OSSCallbackInput{
		Callback:    `{"callbackUrl":"https://callback.example/upload","callbackBody":"bucket=${bucket}&object=${object}&size=${size}&sha1=${sha1}","callbackBodyType":"application/x-www-form-urlencoded"}`,
		CallbackVar: `{"x:keep":"yes"}`,
		Bucket:      "bucket-1",
		Object:      "object-1",
		FileSize:    1024,
		FileSha1:    "SHA1",
	})
	if err != nil {
		t.Fatalf("构造 callback header 失败：%v", err)
	}

	callbackJSON := decodeBase64JSON(t, headers.Callback)
	if callbackJSON["callbackUrl"] != "https://callback.example/upload" {
		t.Fatalf("callbackUrl = %v", callbackJSON["callbackUrl"])
	}
	if callbackJSON["callbackBodyType"] != "application/x-www-form-urlencoded" {
		t.Fatalf("callbackBodyType = %v", callbackJSON["callbackBodyType"])
	}

	callbackVarJSON := decodeBase64JSON(t, headers.CallbackVar)
	for key, want := range map[string]string{
		"bucket": "bucket-1",
		"object": "object-1",
		"size":   "1024",
		"sha1":   "SHA1",
		"x:keep": "yes",
	} {
		if got := callbackVarJSON[key]; got != want {
			t.Fatalf("callback var %s = %v，期望 %s", key, got, want)
		}
	}
}

func TestParseCompleteCallbackResult(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		want    UploadCompleteResult
		wantErr bool
	}{
		{
			name: "合法回调结果",
			input: map[string]any{
				"state":   true,
				"message": "",
				"data": map[string]any{
					"file_id":   "file-1",
					"pick_code": "pick-1",
					"parent_id": "100",
					"sha1":      "SHA1",
					"size":      float64(1024),
					"mtime":     float64(123456),
				},
			},
			want: UploadCompleteResult{
				FileId:   "file-1",
				PickCode: "pick-1",
				ParentId: "100",
				Sha1:     "SHA1",
				Size:     1024,
				Mtime:    123456,
			},
		},
		{
			name: "callback state false 时失败",
			input: map[string]any{
				"state":   false,
				"message": "callback failed",
			},
			wantErr: true,
		},
		{
			name: "缺少 file_id 时失败",
			input: map[string]any{
				"state":   true,
				"message": "",
				"data": map[string]any{
					"pick_code": "pick-1",
				},
			},
			wantErr: true,
		},
		{
			name: "缺少 pick_code 时失败",
			input: map[string]any{
				"state":   true,
				"message": "",
				"data": map[string]any{
					"file_id": "file-1",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCompleteCallbackResult(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("期望解析失败，实际返回 nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("解析 complete callback 失败：%v", err)
			}
			if got != tt.want {
				t.Fatalf("结果 = %+v，期望 %+v", got, tt.want)
			}
		})
	}
}

func TestOpenClient_UploadResumeUsesOfficialFields(t *testing.T) {
	transport := newCaptureOpenAPITransport(`{"state":true,"code":0,"data":{"pick_code":"pick-2","target":"U_1_100","version":"1","bucket":"bucket-1","object":"object-1","callback":{"callback":"{}","callback_var":"{}"}}}`)
	client := newTestOpenClient(transport)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	got, err := client.UploadResume(ctx, "pick-1", 5335, "100", "SHA1")
	if err != nil {
		t.Fatalf("调用断点续传失败：%v", err)
	}
	if got.PickCode != "pick-2" || got.Bucket != "bucket-1" || got.Object != "object-1" {
		t.Fatalf("断点续传结果 = %+v", got)
	}

	req := receiveCapturedRequest(t, transport)
	if req.Method != http.MethodPost {
		t.Fatalf("请求方法 = %s，want %s", req.Method, http.MethodPost)
	}
	if gotPath := mustParseURLPath(t, req.URL); gotPath != "/open/upload/resume" {
		t.Fatalf("请求路径 = %s，want /open/upload/resume", gotPath)
	}
	form, err := url.ParseQuery(req.Body)
	if err != nil {
		t.Fatalf("解析请求表单失败：%v", err)
	}
	for key, want := range map[string]string{
		"file_size": "5335",
		"target":    "U_1_100",
		"fileid":    "SHA1",
		"pick_code": "pick-1",
	} {
		if gotValue := form.Get(key); gotValue != want {
			t.Fatalf("form[%s] = %s，期望 %s", key, gotValue, want)
		}
	}
}

func TestOpenClient_UploadResumeReturnsOpenAPIError(t *testing.T) {
	transport := newCaptureOpenAPITransport(`{"state":false,"code":990001,"message":"resume failed","data":{}}`)
	client := newTestOpenClient(transport)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.UploadResume(ctx, "pick-1", 5335, "100", "SHA1")
	if err == nil {
		t.Fatal("期望断点续传返回错误，实际为 nil")
	}
}

func TestWaitForRapidUpload(t *testing.T) {
	first := &UploadInitResult{Status: UploadInitStatusNeedUpload, PickCode: "pick-1"}

	t.Run("等待命中秒传", func(t *testing.T) {
		calls := 0
		outcome, err := WaitForRapidUpload(
			context.Background(),
			first,
			RapidUploadWaitPolicy{Enabled: true, Timeout: 2 * time.Second, Interval: time.Second},
			1024,
			func(context.Context) (*UploadInitResult, error) {
				calls++
				if calls == 1 {
					return &UploadInitResult{Status: UploadInitStatusNeedUpload, PickCode: "pick-2"}, nil
				}
				return &UploadInitResult{Status: UploadInitStatusRapidUploaded, FileId: "file-1"}, nil
			},
			func(context.Context, time.Duration) error { return nil },
		)
		if err != nil {
			t.Fatalf("等待秒传失败：%v", err)
		}
		if outcome.Result.FileId != "file-1" || outcome.Attempts != 2 || outcome.TimedOut {
			t.Fatalf("等待结果 = %+v", outcome)
		}
	})

	t.Run("小于最小大小时不等待", func(t *testing.T) {
		calls := 0
		outcome, err := WaitForRapidUpload(
			context.Background(),
			first,
			RapidUploadWaitPolicy{Enabled: true, Timeout: 2 * time.Second, Interval: time.Second, MinSize: 10 * 1024},
			1024,
			func(context.Context) (*UploadInitResult, error) {
				calls++
				return &UploadInitResult{Status: UploadInitStatusRapidUploaded, FileId: "file-1"}, nil
			},
			func(context.Context, time.Duration) error { return nil },
		)
		if err != nil {
			t.Fatalf("等待秒传失败：%v", err)
		}
		if calls != 0 || outcome.Result != first || outcome.Attempts != 0 {
			t.Fatalf("不应等待，calls=%d outcome=%+v", calls, outcome)
		}
	})

	t.Run("超时后按配置跳过真实上传", func(t *testing.T) {
		outcome, err := WaitForRapidUpload(
			context.Background(),
			first,
			RapidUploadWaitPolicy{Enabled: true, Timeout: time.Second, Interval: time.Second, SkipUpload: true},
			1024,
			func(context.Context) (*UploadInitResult, error) {
				return &UploadInitResult{Status: UploadInitStatusNeedUpload, PickCode: "pick-2"}, nil
			},
			func(context.Context, time.Duration) error { return nil },
		)
		if err != nil {
			t.Fatalf("等待秒传失败：%v", err)
		}
		if !outcome.TimedOut || !outcome.SkipUpload || outcome.Attempts != 1 {
			t.Fatalf("超时结果 = %+v", outcome)
		}
	})
}

func decodeBase64JSON(t *testing.T, value string) map[string]string {
	t.Helper()

	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("base64 解码失败：%v", err)
	}
	var data map[string]string
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("JSON 解码失败：%v", err)
	}
	return data
}
