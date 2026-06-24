package helpers

import (
	"fmt"
	"net/url"
	"strings"
)

func MakeOpenListUrl(baseUrl, sign, fileId string) string {
	// 去掉 baseURL 末尾的 /
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	// 去掉 fileId 首尾的 /
	fileId = strings.Trim(fileId, "/")
	// 把 fileId 用 / 分隔，分隔后的每一段都做 URL 编码
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
