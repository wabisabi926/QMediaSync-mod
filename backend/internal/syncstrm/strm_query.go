package syncstrm

import (
	"net/url"
	"strings"
)

func encodeStrmQuery(params url.Values) string {
	return strings.ReplaceAll(params.Encode(), "+", "%20")
}
