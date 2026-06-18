package parallels_test

import (
	"fmt"
	"runtime"
	"testing"

	"Q115-STRM/emby302/util/parallels"
)

func TestSliceChunk(t *testing.T) {
	tests := []int{0, 1, 5, 17}
	for _, size := range tests {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			gotRanges := parallels.SliceChunk(size)
			if size <= 0 {
				if len(gotRanges) != 0 {
					t.Fatalf("SliceChunk(%d) = %v, want empty ranges", size, gotRanges)
				}
				return
			}

			wantLen := min(runtime.NumCPU(), size)
			if len(gotRanges) != wantLen {
				t.Fatalf("SliceChunk(%d) returned %d ranges, want %d: %v", size, len(gotRanges), wantLen, gotRanges)
			}

			previousEnd := 0
			minChunkSize := size
			maxChunkSize := 0
			for _, gotRange := range gotRanges {
				if gotRange.Start != previousEnd {
					t.Fatalf("SliceChunk(%d) returned non-contiguous ranges: %v", size, gotRanges)
				}
				if gotRange.Start >= gotRange.End {
					t.Fatalf("SliceChunk(%d) returned empty or reversed range: %v", size, gotRanges)
				}
				if gotRange.End > size {
					t.Fatalf("SliceChunk(%d) returned out-of-bound range: %v", size, gotRanges)
				}

				chunkSize := gotRange.End - gotRange.Start
				minChunkSize = min(minChunkSize, chunkSize)
				maxChunkSize = max(maxChunkSize, chunkSize)
				previousEnd = gotRange.End
			}
			if previousEnd != size {
				t.Fatalf("SliceChunk(%d) covered [0,%d), want [0,%d): %v", size, previousEnd, size, gotRanges)
			}
			if maxChunkSize-minChunkSize > 1 {
				t.Fatalf("SliceChunk(%d) returned unbalanced ranges: %v", size, gotRanges)
			}
		})
	}
}
