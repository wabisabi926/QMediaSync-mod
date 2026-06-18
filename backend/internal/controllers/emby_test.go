package controllers

import (
	"testing"
)

func TestFormatSeasonEpisodes(t *testing.T) {
	tests := []struct {
		name           string
		seasons        map[int][]int
		expectedResult string
	}{
		{
			name:           "空seasons",
			seasons:        map[int][]int{},
			expectedResult: "",
		},
		{
			name: "单个季单个集",
			seasons: map[int][]int{
				1: {1},
			},
			expectedResult: "S1E1",
		},
		{
			name: "单个季多个连续集",
			seasons: map[int][]int{
				1: {1, 2, 3, 4, 5},
			},
			expectedResult: "S1E1-E5",
		},
		{
			name: "单个季多个不连续集",
			seasons: map[int][]int{
				1: {1, 3, 5, 7},
			},
			expectedResult: "S1E1, E3, E5, E7",
		},
		{
			name: "单个季混合连续和不连续集",
			seasons: map[int][]int{
				1: {1, 2, 3, 5, 6, 8, 10, 11, 12},
			},
			expectedResult: "S1E1-E3, E5-E6, E8, E10-E12",
		},
		{
			name: "多个季各包含连续集",
			seasons: map[int][]int{
				1: {1, 2, 3},
				2: {1, 2, 3, 4},
			},
			expectedResult: "S1E1-E3; S2E1-E4",
		},
		{
			name: "多个季混合情况",
			seasons: map[int][]int{
				1: {1, 2, 3, 5, 7},
				2: {1, 3, 5},
				3: {10, 11, 12, 13},
			},
			expectedResult: "S1E1-E3, E5, E7; S2E1, E3, E5; S3E10-E13",
		},
		{
			name: "季号乱序输入",
			seasons: map[int][]int{
				3: {1, 2, 3},
				1: {1, 2},
				2: {1},
			},
			expectedResult: "S1E1-E2; S2E1; S3E1-E3",
		},
		{
			name: "集号乱序输入",
			seasons: map[int][]int{
				1: {9, 5, 3, 1, 8, 4},
			},
			expectedResult: "S1E1, E3-E5, E8-E9",
		},
		{
			name: "实际场景模拟",
			seasons: map[int][]int{
				1: {1, 2, 3, 4, 5, 6, 7, 8, 9},
				2: {1, 2, 3},
			},
			expectedResult: "S1E1-E9; S2E1-E3",
		},
		{
			name: "单个季只有两个连续集",
			seasons: map[int][]int{
				1: {1, 2},
			},
			expectedResult: "S1E1-E2",
		},
		{
			name: "特殊季号0",
			seasons: map[int][]int{
				0: {1, 2, 3},
			},
			expectedResult: "S0E1-E3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSeasonEpisodes(tt.seasons)
			if result != tt.expectedResult {
				t.Errorf("formatSeasonEpisodes() = %v, 期望 %v", result, tt.expectedResult)
			}
		})
	}
}
