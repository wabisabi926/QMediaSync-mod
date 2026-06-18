// updater/downloader.go
package updater

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DownloadAndInstall 下载并安装更新
func (g *GitHubUpdater) DownloadAndInstall(info *UpdateInfo, targetDir string) error {
	if info.DownloadURL == "" {
		return fmt.Errorf("no download URL available")
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 下载文件
	downloadedFile, err := g.downloadFile(info.DownloadURL, tempDir)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// 验证文件完整性
	if verr := g.verifyFile(downloadedFile, info.Checksum); verr != nil {
		return fmt.Errorf("verification failed: %w", verr)
	}

	// 提取文件
	binaryPath, err := g.extractFile(downloadedFile, tempDir)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// 安装到目标目录
	if err := g.installBinary(binaryPath, targetDir); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	return nil
}

// downloadFile 下载文件
func (g *GitHubUpdater) downloadFile(url, tempDir string) (string, error) {
	resp, err := g.HTTPClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 从 URL 提取文件名
	filename := filepath.Base(url)
	if strings.Contains(filename, "?") {
		filename = strings.Split(filename, "?")[0]
	}

	filePath := filepath.Join(tempDir, filename)

	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// 显示下载进度
	reader := &ProgressReader{
		Reader:   resp.Body,
		Total:    resp.ContentLength,
		Callback: g.printProgress,
	}

	if _, err := io.Copy(out, reader); err != nil {
		return "", err
	}

	fmt.Printf("\nDownload completed: %s\n", filename)
	return filePath, nil
}

// verifyFile 验证文件完整性
func (g *GitHubUpdater) verifyFile(filePath, checksumURL string) error {
	if checksumURL == "" {
		fmt.Println("Warning: No checksum available, skipping verification")
		return nil
	}

	// 下载校验和文件
	checksums, err := g.downloadChecksums(checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}

	// 计算文件哈希
	fileHash, err := g.calculateFileHash(filePath)
	if err != nil {
		return err
	}

	// 查找匹配的校验和
	filename := filepath.Base(filePath)
	expectedHash, found := checksums[filename]
	if !found {
		return fmt.Errorf("no checksum found for file: %s", filename)
	}

	if fileHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, fileHash)
	}

	fmt.Println("File verification passed")
	return nil
}

// downloadChecksums 下载校验和文件
func (g *GitHubUpdater) downloadChecksums(checksumURL string) (map[string]string, error) {
	resp, err := g.HTTPClient.Get(checksumURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	checksums := make(map[string]string)
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 2 {
			checksums[parts[1]] = parts[0] // hash -> filename
		}
	}

	return checksums, scanner.Err()
}

// calculateFileHash 计算文件哈希
func (g *GitHubUpdater) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// extractFile 提取压缩文件
func (g *GitHubUpdater) extractFile(archivePath, tempDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return g.extractZip(archivePath, tempDir)
	} else if strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz") {
		return g.extractTarGz(archivePath, tempDir)
	}

	// 如果不是压缩文件，直接返回
	return archivePath, nil
}

// extractZip 解压 zip 文件
func (g *GitHubUpdater) extractZip(archivePath, tempDir string) (string, error) {
	// zip 解压实现
	return "", nil
}

// extractTarGz 解压 tar.gz 文件
func (g *GitHubUpdater) extractTarGz(archivePath, tempDir string) (string, error) {
	// tar.gz 解压实现
	return "", nil
}

// installBinary 安装二进制文件
func (g *GitHubUpdater) installBinary(binaryPath, targetDir string) error {
	// 获取当前可执行文件路径
	currentExe, err := os.Executable()
	if err != nil {
		return err
	}

	// 创建备份
	backupPath := currentExe + ".bak"
	if err := os.Rename(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// 复制新文件
	targetPath := filepath.Join(targetDir, filepath.Base(currentExe))
	if err := g.copyFile(binaryPath, targetPath); err != nil {
		// 恢复备份
		os.Rename(backupPath, currentExe)
		return fmt.Errorf("failed to install new version: %w", err)
	}

	// 设置可执行权限
	if err := os.Chmod(targetPath, 0777); err != nil {
		return err
	}

	fmt.Printf("Update installed successfully: %s\n", targetPath)
	return nil
}

// copyFile 复制文件
func (g *GitHubUpdater) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// ProgressReader 带进度显示的文件读取器
type ProgressReader struct {
	Reader   io.Reader
	Total    int64
	ByteRead int64
	Callback func(int64, int64)
}

func (r *ProgressReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.ByteRead += int64(n)

	if r.Callback != nil {
		r.Callback(r.ByteRead, r.Total)
	}

	return n, err
}

func (g *GitHubUpdater) printProgress(read, total int64) {
	if total == 0 {
		return
	}

	percent := float64(read) / float64(total) * 100
	fmt.Printf("\rDownloading: %.1f%% [%s/%s]", percent, formatBytes(read), formatBytes(total))
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
