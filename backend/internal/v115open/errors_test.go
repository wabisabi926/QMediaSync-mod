package v115open

import "testing"

func TestOpenAPIErrorIncludesOriginalCodeAndMessage(t *testing.T) {
	err := NewOpenAPIError(40140106, "App ID 无效")

	if err.Error() != "115接口错误(40140106): App ID 无效" {
		t.Fatalf("错误信息未保留原始 code 和 message: %q", err.Error())
	}
}
