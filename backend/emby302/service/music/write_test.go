package music_test

import (
	"os"
	"testing"

	"qmediasync/emby302/service/lib/ffmpeg"
	"qmediasync/emby302/service/music"
	"qmediasync/internal/helpers"
)

const host = "http://0.0.0.0:12345"

func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("QMS_INTEGRATION_TESTS") != "1" {
		t.Skip("设置 QMS_INTEGRATION_TESTS=1 后运行音乐集成测试")
	}
	helpers.ConfigDir = t.TempDir()
	helpers.AppLogger = helpers.NewLogger("test.log", true, false)
	t.Cleanup(func() {
		helpers.AppLogger.Close()
		helpers.AppLogger = nil
	})
}

func TestWriteFakeMP3(t *testing.T) {
	requireIntegration(t)
	if err := ffmpeg.AutoDownloadExec("../../.."); err != nil {
		t.Fatal(err)
		return
	}

	url := host + "/d/%E9%9F%B3%E4%B9%90/%E5%BE%90%E4%BD%B3%E8%8E%B9%E3%80%81%E9%99%88%E6%A5%9A%E7%94%9F%20-%20%E8%BA%AB%E9%AA%91%E7%99%BD%E9%A9%AC%20(Live).flac?sign=A6lbwyyDt8g2bsWB3GgfZ2aC1VGklJ2puYv_EZAJsLM=:0"
	meta, err := ffmpeg.InspectMusic(url)
	if err != nil {
		t.Fatal(err)
		return
	}
	cover, err := ffmpeg.ExtractMusicCover(url)
	if err != nil {
		t.Fatal(err)
		return
	}
	path := "../../../openlist-local-tree/音乐"
	os.MkdirAll(path, os.ModePerm)
	err = music.WriteFakeMP3(path+"/fake.flac", meta, cover)
	if err != nil {
		t.Fatal(err)
	}
}
