package syncstrm

import (
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	"qmediasync/internal/models"
	"qmediasync/internal/v115open"
)

// 启动百度网盘同步
// 如果有全量标识，走 StartOther（递归文件夹）。
// 如果没有全量标识：有 LastSyncAt 时从 LastSyncAt 开始同步（递归接口），否则走 StartOther。
func (s *SyncStrm) StartBaiduPanSync() {
	if !s.TmpSyncPath {
		// 如果是今天的第一次同步，则执行全量同步
		// 根据 SyncPathId 查询今天是否有同步记录。
		// 如果有，从 LastSyncAt 开始同步（走递归接口）。
		// 如果没有，执行全量同步（走递归文件夹）
		sync := models.GetTodayFirstSyncByPathId(s.SyncPathId)
		if sync == nil {
			s.FullSync = true
		}
	}
	s.Sync.Logger.Infof("最后同步时间：%d", s.LastSyncAt)
	if s.FullSync || s.LastSyncAt == 0 {
		s.Sync.Logger.Infof("执行百度网盘全量同步")
		s.StartOther()
		return
	}
	// 使用增量同步
	if s.LastSyncAt != 0 {
		s.Sync.Logger.Infof("从修改时间 %d 开始增量同步", s.LastSyncAt)
		err := s.StartBaiduPanSyncByMtime(s.LastSyncAt)
		if err != nil {
			s.Sync.Logger.Errorf("从修改时间 %d 开始同步失败：%v", s.LastSyncAt, err)
			s.PathErrChan <- err
			return
		}
		// 将数据库中的数据加入到缓存中，供后续使用
		s.LoadSyncFileToCache()
		return
	}
	s.Sync.Logger.Infof("执行百度网盘全量同步")
	// 非全量且没有 LastSyncAt，执行全量同步。
	s.StartOther()
}

func (s *SyncStrm) StartBaiduPanSyncByMtime(lastSyncAt int64) error {
	// 从 lastSyncAt 开始同步
	offset := 0
	reqCount := 0
mainloop:
	for {
		if reqCount > 8 {
			// 每 8 次，休息 1 分钟
			time.Sleep(60 * time.Second)
			reqCount = 0
		}
		select {
		case <-s.Context.Done():
			return s.Context.Err()
		default:
			fileListResp, err := s.SyncDriver.GetFilesByPathMtime(s.Context, s.SourcePath, offset, 1000, lastSyncAt)
			reqCount++
			if err != nil {
				s.Sync.Logger.Errorf("同步修改时间 %d 之后的文件失败，offset=%d，错误：%v", lastSyncAt, offset, err)
				s.PathErrChan <- err
				return err
			}
			for _, file := range fileListResp.List {
				atomic.AddInt64(&s.TotalFile, 1)
				s.PublishProgress(false)
				if s.IsExcludePath(file.Path) {
					s.Sync.Logger.Warnf("文件路径 %s 中有排除项，被排除", file.Path)
					continue
				}
				parentPath := filepath.ToSlash(filepath.Dir(file.Path))
				syncFile := SyncFileCache{
					ParentId:   parentPath,
					FileId:     file.Path,
					PickCode:   fmt.Sprintf("%d", file.FsId),
					Path:       parentPath,
					FileName:   filepath.Base(file.Path),
					FileType:   v115open.TypeFile,
					FileSize:   int64(file.Size),
					MTime:      int64(file.ServerMtime),
					Sha1:       file.Md5,
					SourceType: models.SourceTypeBaiduPan,
				}
				// s.Sync.Logger.Infof("文件 %s => %s 路径 %s", syncFile.FileId, syncFile.FileName, syncFile.LocalFilePath)
				if file.IsDir == 1 {
					syncFile.FileType = v115open.TypeDir
					syncFile.IsVideo = false
					syncFile.IsMeta = false
					syncFile.GetLocalFilePath(s.TargetPath, s.SourcePath) // 生成本地路径缓存
				} else {
					if !s.ValidFile(&syncFile) {
						continue
					}
					syncFile.GetLocalFilePath(s.TargetPath, s.SourcePath) // 生成本地路径缓存
				}
				// 放入同步缓存
				err := s.memSyncCache.Insert(&syncFile)
				if err != nil {
					s.Sync.Logger.Errorf("文件 %s => %s 插入同步缓存失败：%v", syncFile.FileId, syncFile.FileName, err)
					return err
				}
				// s.Sync.Logger.Infof("文件 %s => %s 插入同步缓存成功，路径 %s", syncFile.FileId, syncFile.FileName, syncFile.LocalFilePath)
				// 如果路径完整，直接处理文件
				if syncFile.LocalFilePath != "" {
					s.processNetFile(&syncFile)
				}
			}
			offset = int(fileListResp.Cursor)
			if fileListResp.HasMore == 0 || len(fileListResp.List) < 1000 {
				break mainloop
			}
		}
	}
	return nil
}

func (s *SyncStrm) LoadSyncFileToCache() {
	s.Sync.Logger.Infof("从数据库中查询上次同步的文件")
	offset := 0
	limit := 1000
	for {
		syncFiles, err := models.GetFilesBySyncPathId(s.SyncPathId, offset, limit)
		if err != nil {
			s.Sync.Logger.Errorf("从数据库中查询上次同步的文件失败，offset=%d，错误：%v", offset, err)
			return
		}
		if len(syncFiles) == 0 {
			s.Sync.Logger.Infof("从数据库中查询上次同步的文件，offset=%d 没有更多数据", offset)
			break
		}
		for _, item := range syncFiles {
			syncFileCache := SyncFileCache{
				ParentId:      item.ParentId,
				FileId:        item.FileId,
				PickCode:      item.PickCode,
				Path:          item.Path,
				FileName:      item.FileName,
				FileType:      item.FileType,
				FileSize:      item.FileSize,
				MTime:         item.MTime,
				Sha1:          item.Sha1,
				SourceType:    item.SourceType,
				LocalFilePath: item.LocalFilePath,
				IsVideo:       item.IsVideo,
				IsMeta:        item.IsMeta,
			}
			err := s.memSyncCache.Insert(&syncFileCache)
			if err != nil {
				s.Sync.Logger.Errorf("文件 %s => %s 插入同步缓存失败：%v", syncFileCache.FileId, syncFileCache.FileName, err)
				return
			} else {
				// s.Sync.Logger.Infof("文件 %s => %s 插入同步缓存成功，路径 %s", syncFileCache.FileId, syncFileCache.FileName, syncFileCache.LocalFilePath)
			}
		}
		if len(syncFiles) < limit {
			s.Sync.Logger.Infof("从数据库中查询上次同步的文件，offset=%d 没有更多数据", offset)
			break
		}
		offset += limit
	}
}
