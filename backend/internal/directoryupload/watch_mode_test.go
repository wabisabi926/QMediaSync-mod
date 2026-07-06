package directoryupload

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qmediasync/internal/models"
)

type fakeWatchModeDetector struct {
	filesystemType string
	filesystemErr  error
	limits         inotifyLimits
	limitsErr      error
	directoryCount int
	directoryErr   error

	filesystemCalls int
	limitsCalls     int
	directoryCalls  int
}

func (d *fakeWatchModeDetector) FilesystemType(context.Context, string) (string, error) {
	d.filesystemCalls++
	return d.filesystemType, d.filesystemErr
}

func (d *fakeWatchModeDetector) InotifyLimits(context.Context) (inotifyLimits, error) {
	d.limitsCalls++
	return d.limits, d.limitsErr
}

func (d *fakeWatchModeDetector) WatchDirectoryCount(context.Context, *models.DirectoryUploadRule) (int, error) {
	d.directoryCalls++
	return d.directoryCount, d.directoryErr
}

func TestWatchModeDecision(t *testing.T) {
	tests := []struct {
		name             string
		detector         *fakeWatchModeDetector
		wantMode         RuleRuntimeMode
		wantReason       string
		wantLimitsCalls  int
		wantDirCountCall int
	}{
		{
			name: "网络文件系统使用 polling",
			detector: &fakeWatchModeDetector{
				filesystemType: "nfs4",
				limits: inotifyLimits{
					Available:        true,
					MaxUserWatches:   1024,
					MaxUserInstances: 128,
				},
				directoryCount: 1,
			},
			wantMode:         RuleRuntimeModePolling,
			wantReason:       "nfs4",
			wantLimitsCalls:  0,
			wantDirCountCall: 0,
		},
		{
			name: "待 watch 目录数达到阈值使用 polling",
			detector: &fakeWatchModeDetector{
				filesystemType: "ext4",
				limits: inotifyLimits{
					Available:        true,
					MaxUserWatches:   10,
					MaxUserInstances: 128,
				},
				directoryCount: 8,
			},
			wantMode:         RuleRuntimeModePolling,
			wantReason:       "8/10",
			wantLimitsCalls:  1,
			wantDirCountCall: 1,
		},
		{
			name: "普通本地目录使用 watcher",
			detector: &fakeWatchModeDetector{
				filesystemType: "ext4",
				limits: inotifyLimits{
					Available:        true,
					MaxUserWatches:   100,
					MaxUserInstances: 128,
					CurrentWatches:   10,
					CurrentInstances: 1,
				},
				directoryCount: 10,
			},
			wantMode:         RuleRuntimeModeWatcher,
			wantReason:       "20/100",
			wantLimitsCalls:  1,
			wantDirCountCall: 1,
		},
		{
			name: "当前 watch 使用量加本规则目录数达到阈值使用 polling",
			detector: &fakeWatchModeDetector{
				filesystemType: "ext4",
				limits: inotifyLimits{
					Available:        true,
					MaxUserWatches:   100,
					MaxUserInstances: 128,
					CurrentWatches:   70,
					CurrentInstances: 1,
				},
				directoryCount: 10,
			},
			wantMode:         RuleRuntimeModePolling,
			wantReason:       "80/100",
			wantLimitsCalls:  1,
			wantDirCountCall: 1,
		},
		{
			name: "当前 instance 使用量加本规则 watcher 达到阈值使用 polling",
			detector: &fakeWatchModeDetector{
				filesystemType: "ext4",
				limits: inotifyLimits{
					Available:        true,
					MaxUserWatches:   100,
					MaxUserInstances: 10,
					CurrentWatches:   1,
					CurrentInstances: 7,
				},
				directoryCount: 1,
			},
			wantMode:         RuleRuntimeModePolling,
			wantReason:       "8/10",
			wantLimitsCalls:  1,
			wantDirCountCall: 1,
		},
		{
			name: "检测失败不阻塞 fsnotify",
			detector: &fakeWatchModeDetector{
				filesystemErr: errors.New("mountinfo unavailable"),
				limitsErr:     errors.New("proc unavailable"),
			},
			wantMode:         RuleRuntimeModeWatcher,
			wantReason:       "继续尝试 fsnotify",
			wantLimitsCalls:  1,
			wantDirCountCall: 0,
		},
		{
			name: "非 Linux 环境不统计 inotify 目录",
			detector: &fakeWatchModeDetector{
				filesystemType: "apfs",
				limits:         inotifyLimits{Available: false},
			},
			wantMode:         RuleRuntimeModeWatcher,
			wantReason:       "非 Linux",
			wantLimitsCalls:  1,
			wantDirCountCall: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &models.DirectoryUploadRule{
				MonitorPath: "/media",
				Recursive:   true,
			}

			got := decideWatchMode(context.Background(), rule, tt.detector)

			if got.Mode != tt.wantMode {
				t.Fatalf("mode=%s，期望 %s，reason=%s", got.Mode, tt.wantMode, got.Reason)
			}
			if !strings.Contains(got.Reason, tt.wantReason) {
				t.Fatalf("reason=%q，期望包含 %q", got.Reason, tt.wantReason)
			}
			if tt.detector.limitsCalls != tt.wantLimitsCalls {
				t.Fatalf("limits calls=%d，期望 %d", tt.detector.limitsCalls, tt.wantLimitsCalls)
			}
			if tt.detector.directoryCalls != tt.wantDirCountCall {
				t.Fatalf("directory calls=%d，期望 %d", tt.detector.directoryCalls, tt.wantDirCountCall)
			}
		})
	}
}

func TestFilesystemTypeFromMountInfo(t *testing.T) {
	tests := []struct {
		name        string
		monitorPath string
		mountInfo   string
		wantType    string
		wantOK      bool
	}{
		{
			name:        "根挂载和子挂载取最长匹配",
			monitorPath: "/mnt/media/show",
			mountInfo: strings.Join([]string{
				"1 0 0:1 / / rw - ext4 /dev/root rw",
				"2 1 0:2 / /mnt rw - xfs /dev/sdb1 rw",
				"3 2 0:3 / /mnt/media rw - nfs4 server:/media rw",
			}, "\n"),
			wantType: "nfs4",
			wantOK:   true,
		},
		{
			name:        "路径前缀不是子挂载不匹配",
			monitorPath: "/mnt/foobar/movie",
			mountInfo:   "1 0 0:1 / /mnt/foo rw - ext4 /dev/sdb1 rw",
			wantOK:      false,
		},
		{
			name:        "mountinfo 转义空格路径",
			monitorPath: "/mnt/foo bar/movie",
			mountInfo:   `1 0 0:1 / /mnt/foo\040bar rw - ext4 /dev/sdb1 rw`,
			wantType:    "ext4",
			wantOK:      true,
		},
		{
			name:        "未匹配路径",
			monitorPath: "/missing/movie",
			mountInfo:   "1 0 0:1 / /mnt/media rw - ext4 /dev/sdb1 rw",
			wantOK:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotOK := filesystemTypeFromMountInfo(tt.monitorPath, tt.mountInfo)
			if gotOK != tt.wantOK {
				t.Fatalf("ok=%v，期望 %v，filesystem=%q", gotOK, tt.wantOK, gotType)
			}
			if gotType != tt.wantType {
				t.Fatalf("filesystem=%q，期望 %q", gotType, tt.wantType)
			}
		})
	}
}

func TestWatchModeDecisionCountsDirectoriesOnly(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Show", "Season 01"), 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "movie.mkv"), []byte("movie"), 0o644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "Show", "Season 01", "episode.mkv"), []byte("episode"), 0o644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	detector := newOSWatchModeDetectorForGOOS("linux")
	recursiveRule := &models.DirectoryUploadRule{MonitorPath: root, Recursive: true}
	got, err := detector.WatchDirectoryCount(context.Background(), recursiveRule)
	if err != nil {
		t.Fatalf("统计递归 watch 目录失败: %v", err)
	}
	if got != 3 {
		t.Fatalf("recursive directory count=%d，期望只统计 3 个目录", got)
	}

	flatRule := &models.DirectoryUploadRule{MonitorPath: root, Recursive: false}
	got, err = detector.WatchDirectoryCount(context.Background(), flatRule)
	if err != nil {
		t.Fatalf("统计非递归 watch 目录失败: %v", err)
	}
	if got != 1 {
		t.Fatalf("flat directory count=%d，期望只统计根目录", got)
	}
}

func TestWatchModeDecisionSkipsIgnoredDirectoriesInWatchCount(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "Show", "Season 01"),
		filepath.Join(root, "Cache", "Season 01"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("创建测试目录失败: %v", err)
		}
	}

	detector := newOSWatchModeDetectorForGOOS("linux")
	rule := &models.DirectoryUploadRule{
		MonitorPath:       root,
		Recursive:         true,
		IgnorePatternsStr: `["Cache"]`,
	}
	got, err := detector.WatchDirectoryCount(context.Background(), rule)
	if err != nil {
		t.Fatalf("统计 watch 目录失败: %v", err)
	}
	if got != 3 {
		t.Fatalf("watch directory count=%d，期望 ignored 子目录不计入，只统计根目录和 Show 子树", got)
	}
}

func TestCurrentInotifyUsageCountsOnlyAnonInotifyFDTargets(t *testing.T) {
	procRoot := t.TempDir()
	writeProcFDInfoForTest(t, procRoot, "1", strings.Join([]string{
		"pos:\t0",
		"inotify wd:1 ino:1 sdev:1 mask:2 ignored_mask:0 fhandle-bytes:8 fhandle-type:1 f_handle:1",
		"inotify wd:2 ino:2 sdev:1 mask:2 ignored_mask:0 fhandle-bytes:8 fhandle-type:1 f_handle:2",
	}, "\n"))
	writeProcFDSymlinkForTest(t, procRoot, "1", "anon_inode:inotify")
	writeProcFDInfoForTest(t, procRoot, "2", "pos:\t0\nflags:\t02000000\n")
	writeProcFDSymlinkForTest(t, procRoot, "2", "anon_inode:inotify")
	writeProcFDInfoForTest(t, procRoot, "3", "pos:\t0\nflags:\t02000000\n")
	writeProcFDSymlinkForTest(t, procRoot, "3", filepath.Join(procRoot, "plain-inotify-path"))
	writeProcFDInfoForTest(t, procRoot, "4", "pos:\t0\nflags:\t02000000\n")
	writeProcFDSymlinkForTest(t, procRoot, "4", "socket:[12345]")

	detector := &osWatchModeDetector{goos: "linux", procRoot: procRoot}
	watches, instances, err := detector.currentInotifyUsage()
	if err != nil {
		t.Fatalf("统计当前 inotify 使用量失败: %v", err)
	}
	if watches != 2 {
		t.Fatalf("current watches=%d，期望 2", watches)
	}
	if instances != 2 {
		t.Fatalf("current instances=%d，期望只统计 anon_inode:inotify fd", instances)
	}
}

func TestInotifyLimitsReadsProcRootAndCurrentUsage(t *testing.T) {
	procRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(procRoot, "sys", "fs", "inotify"), 0o755); err != nil {
		t.Fatalf("创建 inotify proc 目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "sys", "fs", "inotify", "max_user_watches"), []byte("128\n"), 0o644); err != nil {
		t.Fatalf("写入 max_user_watches 失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "sys", "fs", "inotify", "max_user_instances"), []byte("64\n"), 0o644); err != nil {
		t.Fatalf("写入 max_user_instances 失败: %v", err)
	}
	writeProcFDInfoForTest(t, procRoot, "7", "inotify wd:1 ino:1 sdev:1 mask:2 ignored_mask:0 fhandle-bytes:8 fhandle-type:1 f_handle:1\n")
	writeProcFDSymlinkForTest(t, procRoot, "7", "anon_inode:inotify")

	detector := &osWatchModeDetector{goos: "linux", procRoot: procRoot}
	limits, err := detector.InotifyLimits(context.Background())
	if err != nil {
		t.Fatalf("读取 inotify limits 失败: %v", err)
	}
	if !limits.Available {
		t.Fatal("linux detector 应报告 inotify limits 可用")
	}
	if limits.MaxUserWatches != 128 {
		t.Fatalf("max_user_watches=%d，期望 128", limits.MaxUserWatches)
	}
	if limits.MaxUserInstances != 64 {
		t.Fatalf("max_user_instances=%d，期望 64", limits.MaxUserInstances)
	}
	if limits.CurrentWatches != 1 {
		t.Fatalf("current watches=%d，期望 1", limits.CurrentWatches)
	}
	if limits.CurrentInstances != 1 {
		t.Fatalf("current instances=%d，期望 1", limits.CurrentInstances)
	}
	if limits.CurrentUsageError != nil {
		t.Fatalf("current usage error=%v，期望 nil", limits.CurrentUsageError)
	}
}

func writeProcFDInfoForTest(t *testing.T, procRoot string, fd string, content string) {
	t.Helper()
	fdInfoDir := filepath.Join(procRoot, "self", "fdinfo")
	if err := os.MkdirAll(fdInfoDir, 0o755); err != nil {
		t.Fatalf("创建 fdinfo 目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fdInfoDir, fd), []byte(content), 0o644); err != nil {
		t.Fatalf("写入 fdinfo 失败: %v", err)
	}
}

func writeProcFDSymlinkForTest(t *testing.T, procRoot string, fd string, target string) {
	t.Helper()
	fdDir := filepath.Join(procRoot, "self", "fd")
	if err := os.MkdirAll(fdDir, 0o755); err != nil {
		t.Fatalf("创建 fd 目录失败: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(fdDir, fd)); err != nil {
		t.Skipf("当前平台不支持创建 fd symlink: %v", err)
	}
}
