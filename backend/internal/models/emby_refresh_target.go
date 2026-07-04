package models

import (
	"fmt"
	"sort"

	"qmediasync/internal/db"
	embyclientrestgo "qmediasync/internal/embyclient-rest-go"
	"qmediasync/internal/helpers"

	"gorm.io/gorm"
)

// EmbyRefreshTargetType 是 Emby 刷新目标类型。
type EmbyRefreshTargetType string

const (
	EmbyRefreshTargetTypeLibrary EmbyRefreshTargetType = EmbyLibraryRefreshTargetTypeLibrary
	EmbyRefreshTargetTypeItem    EmbyRefreshTargetType = EmbyLibraryRefreshTargetTypeItem
)

// EmbyRefreshTarget 描述一次 STRM 变更应刷新的 Emby 目标。
type EmbyRefreshTarget struct {
	TargetType          EmbyRefreshTargetType
	ItemID              string
	ItemName            string
	ItemType            string
	Recursive           bool
	SyncPathID          uint
	FallbackLibraryId   string
	FallbackLibraryName string
}

// ResolveEmbyRefreshTarget 解析 SyncFile 对应的 Emby 刷新目标。
func ResolveEmbyRefreshTarget(syncFile *SyncFile) (EmbyRefreshTarget, error) {
	if syncFile == nil {
		return EmbyRefreshTarget{TargetType: EmbyRefreshTargetTypeLibrary}, nil
	}
	fallbackLibraryID, fallbackLibraryName := firstEmbyLibraryForSyncPath(syncFile.SyncPathId)
	withFallback := func(target EmbyRefreshTarget) EmbyRefreshTarget {
		target.SyncPathID = syncFile.SyncPathId
		target.FallbackLibraryId = fallbackLibraryID
		target.FallbackLibraryName = fallbackLibraryName
		return target
	}

	if item, err := findEmbyItemBySyncFile(syncFile); err != nil {
		return EmbyRefreshTarget{}, err
	} else if item != nil {
		return withFallback(targetFromEmbyMediaItem(item)), nil
	}

	if target, ok, err := resolveSiblingSeasonOrSeries(syncFile); err != nil {
		return EmbyRefreshTarget{}, err
	} else if ok {
		return withFallback(target), nil
	}

	if item, err := findEmbyItemByPath(syncFile); err != nil {
		helpers.AppLogger.Warnf("按路径兜底查询 Emby 条目失败，将回退媒体库刷新：%v", err)
	} else if item != nil {
		return withFallback(targetFromEmbyPathItem(item)), nil
	}

	return withFallback(EmbyRefreshTarget{TargetType: EmbyRefreshTargetTypeLibrary}), nil
}

func findEmbyItemBySyncFile(syncFile *SyncFile) (*EmbyMediaItem, error) {
	if syncFile == nil {
		return nil, nil
	}
	query := db.Db.Where("sync_file_id = ?", syncFile.ID)
	if syncFile.PickCode != "" {
		query = db.Db.Where("sync_file_id = ? OR pick_code = ?", syncFile.ID, syncFile.PickCode)
	}
	var relation EmbyMediaSyncFile
	err := query.Order("id ASC").First(&relation).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return findEmbyMediaItemByRelation(relation, syncFile)
}

func findEmbyMediaItemByRelation(relation EmbyMediaSyncFile, syncFile *SyncFile) (*EmbyMediaItem, error) {
	var item EmbyMediaItem
	itemID := fmt.Sprintf("%d", relation.EmbyItemId)
	err := db.Db.Where("item_id_int = ? OR item_id = ?", int64(relation.EmbyItemId), itemID).
		Order("id ASC").
		First(&item).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if syncFile != nil && cachedEmbyItemNeedsValidation(&item) {
		exists, err := cachedEmbyItemExistsInEmby(&item)
		if err != nil {
			helpers.AppLogger.Warnf("校验 Emby 缓存条目 %s 失败，继续使用本地索引：%v", item.ItemId, err)
			return &item, nil
		}
		if !exists {
			return nil, nil
		}
	}
	return &item, nil
}

func cachedEmbyItemNeedsValidation(item *EmbyMediaItem) bool {
	return item != nil &&
		item.LastSeenAt == 0 &&
		GlobalEmbyConfig != nil &&
		GlobalEmbyConfig.EmbyUrl != "" &&
		GlobalEmbyConfig.EmbyApiKey != ""
}

func cachedEmbyItemExistsInEmby(item *EmbyMediaItem) (bool, error) {
	if item == nil {
		return false, nil
	}
	itemID := item.ItemId
	if itemID == "" && item.ItemIdInt > 0 {
		itemID = fmt.Sprintf("%d", item.ItemIdInt)
	}
	if itemID == "" {
		return false, nil
	}
	client := embyclientrestgo.NewClient(GlobalEmbyConfig.EmbyUrl, GlobalEmbyConfig.EmbyApiKey)
	found, err := client.FindItemByID(itemID)
	if err != nil {
		return true, err
	}
	return found != nil, nil
}

func resolveSiblingSeasonOrSeries(syncFile *SyncFile) (EmbyRefreshTarget, bool, error) {
	if syncFile == nil || syncFile.Path == "" || syncFile.SyncPathId == 0 {
		return EmbyRefreshTarget{}, false, nil
	}
	var siblings []SyncFile
	query := db.Db.Where("sync_path_id = ? AND path = ?", syncFile.SyncPathId, syncFile.Path)
	if syncFile.ID > 0 {
		query = query.Where("id <> ?", syncFile.ID)
	}
	if err := query.Order("id DESC").Limit(50).Find(&siblings).Error; err != nil {
		return EmbyRefreshTarget{}, false, err
	}
	var seriesTarget EmbyRefreshTarget
	for _, sibling := range siblings {
		item, err := findEmbyItemBySyncFile(&sibling)
		if err != nil {
			return EmbyRefreshTarget{}, false, err
		}
		if item == nil || item.Type != "Episode" {
			continue
		}
		if item.SeasonId != "" {
			return EmbyRefreshTarget{
				TargetType: EmbyRefreshTargetTypeItem,
				ItemID:     item.SeasonId,
				ItemName:   item.SeasonName,
				ItemType:   "Season",
				Recursive:  true,
			}, true, nil
		}
		if item.SeriesId != "" && seriesTarget.ItemID == "" {
			seriesTarget = EmbyRefreshTarget{
				TargetType: EmbyRefreshTargetTypeItem,
				ItemID:     item.SeriesId,
				ItemName:   item.SeriesName,
				ItemType:   "Series",
				Recursive:  true,
			}
		}
	}
	if seriesTarget.ItemID != "" {
		return seriesTarget, true, nil
	}
	return EmbyRefreshTarget{}, false, nil
}

func findEmbyItemByPath(syncFile *SyncFile) (*embyclientrestgo.BaseItemDtoV2, error) {
	if syncFile == nil || syncFile.LocalFilePath == "" || GlobalEmbyConfig == nil || GlobalEmbyConfig.EmbyUrl == "" || GlobalEmbyConfig.EmbyApiKey == "" {
		return nil, nil
	}
	client := embyclientrestgo.NewClient(GlobalEmbyConfig.EmbyUrl, GlobalEmbyConfig.EmbyApiKey)
	return client.FindItemByPath(syncFile.LocalFilePath)
}

func targetFromEmbyMediaItem(item *EmbyMediaItem) EmbyRefreshTarget {
	if item == nil {
		return EmbyRefreshTarget{TargetType: EmbyRefreshTargetTypeLibrary}
	}
	itemID := item.ItemId
	if itemID == "" && item.ItemIdInt > 0 {
		itemID = fmt.Sprintf("%d", item.ItemIdInt)
	}
	return EmbyRefreshTarget{
		TargetType: EmbyRefreshTargetTypeItem,
		ItemID:     itemID,
		ItemName:   item.Name,
		ItemType:   item.Type,
		Recursive:  isRecursiveEmbyRefreshType(item.Type),
	}
}

func targetFromEmbyPathItem(item *embyclientrestgo.BaseItemDtoV2) EmbyRefreshTarget {
	if item == nil {
		return EmbyRefreshTarget{TargetType: EmbyRefreshTargetTypeLibrary}
	}
	return EmbyRefreshTarget{
		TargetType: EmbyRefreshTargetTypeItem,
		ItemID:     item.Id,
		ItemName:   item.Name,
		ItemType:   item.Type,
		Recursive:  isRecursiveEmbyRefreshType(item.Type),
	}
}

func isRecursiveEmbyRefreshType(itemType string) bool {
	switch itemType {
	case "Season", "Series", "Folder":
		return true
	default:
		return false
	}
}

func firstEmbyLibraryForSyncPath(syncPathID uint) (string, string) {
	libraries := GetEmbyLibraryIdsBySyncPathId(syncPathID)
	if len(libraries) == 0 {
		return "", ""
	}
	ids := make([]string, 0, len(libraries))
	for id := range libraries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	firstID := ids[0]
	return firstID, libraries[firstID]
}
