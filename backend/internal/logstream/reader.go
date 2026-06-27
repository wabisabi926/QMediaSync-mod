package logstream

import (
	"bufio"
	"io"
	"os"
)

const (
	defaultTailLimit = 1000
	maxScannerBytes  = 1024 * 1024
)

// ReadTailEntries 读取文件末尾最近 limit 行，并返回文件末尾 cursor。
func ReadTailEntries(path string, limit int) ([]Entry, int64, error) {
	if limit <= 0 {
		limit = defaultTailLimit
	}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, 0, nil
		}
		return nil, 0, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	cursor := stat.Size()
	lines := make([]string, 0, limit)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxScannerBytes)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > limit {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	entries := make([]Entry, 0, len(lines))
	for _, line := range lines {
		entries = append(entries, ParseLine(line))
	}
	return entries, cursor, nil
}

// ReadEndCursor 返回文件末尾 cursor，不读取和解析日志内容。
func ReadEndCursor(path string) (int64, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return stat.Size(), nil
}

// ReadEntriesFromCursor 从 cursor 开始读取完整日志行。
func ReadEntriesFromCursor(path string, cursor int64, limit int) ([]Entry, int64, error) {
	if limit <= 0 {
		limit = defaultTailLimit
	}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, 0, nil
		}
		return nil, 0, err
	}
	defer file.Close()

	if _, err := file.Seek(cursor, io.SeekStart); err != nil {
		return nil, cursor, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxScannerBytes)
	entries := make([]Entry, 0, limit)
	nextCursor := cursor
	for scanner.Scan() && len(entries) < limit {
		line := scanner.Text()
		nextCursor += int64(len(line) + 1)
		entry := ParseLine(line)
		entry.Cursor = nextCursor
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, cursor, err
	}
	return entries, nextCursor, nil
}
