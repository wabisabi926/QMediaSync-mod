package syncstrm

import (
	"net/url"
	"strings"
)

func encodeStrmQuery(params url.Values) string {
	return strings.ReplaceAll(params.Encode(), "+", "%20")
}

func encodeStrmQueryPathLast(params url.Values) string {
	pathValues, hasPath := params["path"]
	if !hasPath || len(pathValues) == 0 {
		return encodeStrmQuery(params)
	}

	otherParams := url.Values{}
	for key, values := range params {
		if key == "path" {
			continue
		}
		otherParams[key] = append([]string(nil), values...)
	}

	encodedQuery := encodeStrmQuery(otherParams)
	encodedPath := encodeStrmQuery(url.Values{"path": append([]string(nil), pathValues...)})
	if encodedQuery == "" {
		return encodedPath
	}
	if encodedPath == "" {
		return encodedQuery
	}
	return encodedQuery + "&" + encodedPath
}

func expectedStrmQueryForSyncFile(mode int, file *SyncFileCache, userID string) string {
	params := url.Values{}
	params.Add("pickcode", file.PickCode)
	params.Add("userid", userID)
	if pathValue := strmPathQueryValue(mode, file); pathValue != "" {
		params.Add("path", pathValue)
	}
	return encodeStrmQueryPathLast(params)
}
