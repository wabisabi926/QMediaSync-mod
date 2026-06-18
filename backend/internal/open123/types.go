package open123

type RespBase[T any] struct {
	XTraceID string `json:"x-traceID"`
	Code     int    `json:"code"`
	Message  string `json:"message"`
	Data     T      `json:"data"`
}

type FileUploadCreateRequest struct {
	ParentFileID int64  `json:"parentFileID"`
	Filename     string `json:"filename"`
	Etag         string `json:"etag"`
	Size         int64  `json:"size"`
}

type FileUploadCreateResponse struct {
	FileID       int64  `json:"fileID"`
	UploadID     string `json:"uploadID"`
	PartSize     int64  `json:"partSize"`
	AlreadyExist bool   `json:"alreadyExist"`
}

type UploadDomainResponse struct {
	Domains []string `json:"data"`
}

type FileInfo struct {
	FileID     int64  `json:"fileID"`
	FileName   string `json:"fileName"`
	FileSize   int64  `json:"fileSize"`
	CreateTime int64  `json:"createTime"`
	UpdateTime int64  `json:"updateTime"`
	FileType   int    `json:"fileType"`
	Etag       string `json:"etag"`
}

type FileListRequest struct {
	DriveID      int64 `json:"driveID"`
	ParentFileID int64 `json:"parentFileID"`
	Page         int   `json:"page"`
	PageSize     int   `json:"pageSize"`
}

type FileListResponse struct {
	Total    int64      `json:"total"`
	NextPage int        `json:"nextPage"`
	FileList []FileInfo `json:"fileList"`
}

type CreateFolderRequest struct {
	DriveID      int64  `json:"driveID"`
	ParentFileID int64  `json:"parentFileID"`
	DirName      string `json:"dirName"`
}

type CreateFolderResponse struct {
	DirID       int64 `json:"dirID"`
	ParentDirID int64 `json:"parentDirID"`
}

type DeleteRequest struct {
	FileID  int64 `json:"fileID"`
	DriveID int64 `json:"driveID"`
}

type DeleteFolderRequest struct {
	DirID   int64 `json:"dirID"`
	DriveID int64 `json:"driveID"`
}

type FileDownloadInfoRequest struct {
	FileID  int64 `json:"fileID" form:"fileID"`
	DriveID int64 `json:"driveID"`
}

type FileDownloadInfoResponse struct {
	FileID      int64  `json:"fileID"`
	FileName    string `json:"fileName"`
	FileSize    int64  `json:"fileSize"`
	DownloadURL string `json:"downloadUrl"`
	ExpireTime  int64  `json:"expireTime"`
}
