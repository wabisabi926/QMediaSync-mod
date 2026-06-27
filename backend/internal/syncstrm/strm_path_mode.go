package syncstrm

func strmPathQueryValue(mode int, file *SyncFileCache) string {
	switch mode {
	case 1:
		return file.GetFullRemotePath()
	case 2:
		return file.FileName
	default:
		return ""
	}
}

func expectedStrmPathForSyncFile(mode int, file *SyncFileCache) string {
	return strmPathQueryValue(mode, file)
}
