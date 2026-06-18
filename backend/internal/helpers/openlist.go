package helpers

import (
	"fmt"
	"net/url"
	"strings"
)

func MakeOpenListUrl(baseUrl, sign, fileId string) string {
	// 去掉BaseUrl末尾的/
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	// 去掉sf.FileId首尾的/
	fileId = strings.Trim(fileId, "/")
	// 把fileId用/分隔，分隔后的每一段都做urlencode
	// 把fileId用/分隔，分隔后的每一段都做urlencode
	parts := strings.Split(fileId, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	fileId = strings.Join(parts, "/")
	url := fmt.Sprintf("%s/d/%s", baseUrl, fileId)
	if sign != "" {
		url += "?sign=" + sign
	}
	return url
}
