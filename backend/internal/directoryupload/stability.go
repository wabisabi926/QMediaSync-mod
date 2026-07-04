package directoryupload

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"qmediasync/internal/models"
)

// StableFile 是已通过稳定性检查的文件。
type StableFile struct {
	Path      string
	Size      int64
	ModTime   time.Time
	Signature string
}

type stabilityCandidate struct {
	path        string
	signature   string
	size        int64
	modTime     time.Time
	stableSince time.Time
	stableCount int
}

// StabilityQueueOptions 是稳定性队列依赖。
type StabilityQueueOptions struct {
	Now  func() time.Time
	Stat func(string) (os.FileInfo, error)
}

// StabilityQueue 跟踪待稳定的本地文件。
type StabilityQueue struct {
	mutex      sync.Mutex
	now        func() time.Time
	stat       func(string) (os.FileInfo, error)
	candidates map[uint]map[string]*stabilityCandidate
}

// NewStabilityQueue 创建稳定性队列。
func NewStabilityQueue(options StabilityQueueOptions) *StabilityQueue {
	now := options.Now
	if now == nil {
		now = time.Now
	}
	stat := options.Stat
	if stat == nil {
		stat = os.Stat
	}
	return &StabilityQueue{
		now:        now,
		stat:       stat,
		candidates: make(map[uint]map[string]*stabilityCandidate),
	}
}

// Track 把文件加入稳定性队列。
func (q *StabilityQueue) Track(ruleID uint, path string) {
	if q == nil || ruleID == 0 || path == "" {
		return
	}
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.candidates[ruleID] == nil {
		q.candidates[ruleID] = make(map[string]*stabilityCandidate)
	}
	if _, ok := q.candidates[ruleID][path]; !ok {
		q.candidates[ruleID][path] = &stabilityCandidate{path: path}
	}
}

// PendingPaths 返回指定规则当前待稳定文件路径。
func (q *StabilityQueue) PendingPaths(ruleID uint) []string {
	if q == nil || ruleID == 0 {
		return []string{}
	}
	q.mutex.Lock()
	defer q.mutex.Unlock()

	paths := make([]string, 0, len(q.candidates[ruleID]))
	for path := range q.candidates[ruleID] {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// Check 执行一次稳定性检查，返回本轮稳定的文件。
func (q *StabilityQueue) Check(rule *models.DirectoryUploadRule) ([]StableFile, error) {
	if q == nil || rule == nil || rule.ID == 0 {
		return []StableFile{}, nil
	}
	q.mutex.Lock()
	defer q.mutex.Unlock()

	ruleCandidates := q.candidates[rule.ID]
	if len(ruleCandidates) == 0 {
		return []StableFile{}, nil
	}

	now := q.now()
	requiredCount := models.DirectoryUploadDefaultStabilityRequiredCount
	requiredDuration := time.Duration(models.DirectoryUploadDefaultStabilitySeconds) * time.Second
	minStableCount := requiredCount
	if requiredDuration <= 0 && minStableCount > 1 {
		minStableCount--
	}
	ready := make([]StableFile, 0)
	for path, candidate := range ruleCandidates {
		info, err := q.stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				delete(ruleCandidates, path)
				continue
			}
			return nil, err
		}
		if info.IsDir() {
			delete(ruleCandidates, path)
			continue
		}
		signature := fileSignature(info)
		if candidate.signature == "" || candidate.signature != signature {
			candidate.signature = signature
			candidate.size = info.Size()
			candidate.modTime = info.ModTime()
			candidate.stableSince = now
			candidate.stableCount = 0
			continue
		}
		candidate.stableCount++
		if candidate.stableCount < minStableCount {
			continue
		}
		if requiredDuration > 0 && now.Sub(candidate.stableSince) < requiredDuration {
			continue
		}
		ready = append(ready, StableFile{
			Path:      path,
			Size:      candidate.size,
			ModTime:   candidate.modTime,
			Signature: candidate.signature,
		})
		delete(ruleCandidates, path)
	}
	sort.Slice(ready, func(i, j int) bool {
		return ready[i].Path < ready[j].Path
	})
	return ready, nil
}

func fileSignature(info os.FileInfo) string {
	if info == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
}
