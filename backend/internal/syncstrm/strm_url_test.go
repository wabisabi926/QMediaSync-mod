package syncstrm

import (
	"net/url"
	"strings"
	"testing"

	"qmediasync/internal/models"
)

func TestMakeStrmContentEncodesPathQuery(t *testing.T) {
	file := &SyncFileCache{
		Path:       "/media/我的朋友很少 (2011)/Season 1",
		FileName:   "我的朋友很少 - S01E03 - 市民泳池没有攻略关键(;’ Д`) + BDRip.mkv",
		PickCode:   "pick-115",
		SourceType: models.SourceType115,
	}
	s := &SyncStrm{
		Config: SyncStrmConfig{
			StrmBaseUrl:     "http://qmediasync:12333",
			StrmUrlNeedPath: 2,
		},
		Account: &models.Account{UserId: "user-115"},
	}
	driver := NewOpen115Driver(nil)
	driver.SetSyncStrm(s)

	content := driver.MakeStrmContent(file)
	parsed, err := url.Parse(content)
	if err != nil {
		t.Fatalf("解析 STRM URL 失败：%v", err)
	}
	values, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		t.Fatalf("解析 STRM query 失败：%v，URL=%s", err, content)
	}

	if got := values.Get("path"); got != file.FileName {
		t.Fatalf("path 参数 = %q，期望 %q", got, file.FileName)
	}
	if strings.Contains(parsed.RawQuery, ";") {
		t.Fatalf("RawQuery 不应包含未编码分号：%s", parsed.RawQuery)
	}
	if strings.Contains(parsed.RawQuery, "+") {
		t.Fatalf("RawQuery 中的空格应编码为 %%20，不应包含 +：%s", parsed.RawQuery)
	}
	if !strings.Contains(parsed.RawQuery, "%20") {
		t.Fatalf("RawQuery 中的空格应编码为 %%20：%s", parsed.RawQuery)
	}
}

func TestBaiduMakeStrmContentEncodesPathQuery(t *testing.T) {
	file := &SyncFileCache{
		Path:       "/media/我的朋友很少 (2011)/Season 1",
		FileName:   "我的朋友很少 - S01E03 - 市民泳池没有攻略关键(;’ Д`) + BDRip.mkv",
		PickCode:   "pick-baidu",
		SourceType: models.SourceTypeBaiduPan,
	}
	s := &SyncStrm{
		Config: SyncStrmConfig{
			StrmBaseUrl:     "http://qmediasync:12333",
			StrmUrlNeedPath: 2,
		},
		Account: &models.Account{UserId: "user-baidu"},
	}
	driver := NewBaiduPanDriver(nil)
	driver.SetSyncStrm(s)

	content := driver.MakeStrmContent(file)
	parsed, err := url.Parse(content)
	if err != nil {
		t.Fatalf("解析 STRM URL 失败：%v", err)
	}
	values, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		t.Fatalf("解析 STRM query 失败：%v，URL=%s", err, content)
	}

	if got := values.Get("path"); got != file.FileName {
		t.Fatalf("path 参数 = %q，期望 %q", got, file.FileName)
	}
	if strings.Contains(parsed.RawQuery, ";") {
		t.Fatalf("RawQuery 不应包含未编码分号：%s", parsed.RawQuery)
	}
	if strings.Contains(parsed.RawQuery, "+") {
		t.Fatalf("RawQuery 中的空格应编码为 %%20，不应包含 +：%s", parsed.RawQuery)
	}
	if !strings.Contains(parsed.RawQuery, "%20") {
		t.Fatalf("RawQuery 中的空格应编码为 %%20：%s", parsed.RawQuery)
	}
}
