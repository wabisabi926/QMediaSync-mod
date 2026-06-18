package emby

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"Q115-STRM/emby302/config"
	"Q115-STRM/emby302/util/https"
	"Q115-STRM/emby302/util/logs"
	"Q115-STRM/emby302/util/strs"

	"Q115-STRM/emby302/util/urls"
	"Q115-STRM/emby302/web/cache"

	"github.com/gin-gonic/gin"
)

// Redirect2Transcode 将 master 请求重定向到本地 ts 代理
func Redirect2Transcode(c *gin.Context) {
	templateId := c.Query("template_id")
	if strs.AnyEmpty(templateId) {
		// 尝试从 mediaSourceId 中获取 templateId
		itemInfo, err := resolveItemInfo(c, RouteTranscode)
		if checkErr(c, err) {
			return
		}
		templateId = itemInfo.MsInfo.TemplateId
	}

	apiKey := c.Query(QueryApiKeyName)
	openlistPath := c.Query("openlist_path")
	if strs.AnyEmpty(templateId) {
		ProxyOrigin(c)
		return
	}

	// 只有 template id 时, 需要先获取 openlist path
	if strs.AnyEmpty(openlistPath) {
		Redirect2OpenlistLink(c)
		return
	}

	tu, _ := url.Parse(https.ClientRequestHost(c.Request) + "/videos/proxy_playlist")
	q := tu.Query()
	q.Set("openlist_path", openlistPath)
	q.Set(QueryApiKeyName, apiKey)
	q.Set("template_id", templateId)
	tu.RawQuery = q.Encode()
	c.Redirect(http.StatusTemporaryRedirect, tu.String())
}

// Redirect2OpenlistLink 重定向资源到 openlist 网盘直链
func Redirect2OpenlistLink(c *gin.Context) {
	// 不处理字幕接口
	if strings.Contains(strings.ToLower(c.Request.RequestURI), "subtitles") {
		ProxyOrigin(c)
		return
	}

	// 1 解析要请求的资源信息
	itemInfo, err := resolveItemInfo(c, RouteStream)
	if checkErr(c, err) {
		return
	}
	// logs.Info("解析到的 itemInfo: %v", itemInfo)

	// 2 如果请求的是转码资源, 重定向到本地的 m3u8 代理服务
	msInfo := itemInfo.MsInfo
	useTranscode := !msInfo.Empty && msInfo.Transcode
	if useTranscode && msInfo.OpenlistPath != "" {
		u, _ := url.Parse(strings.ReplaceAll(MasterM3U8UrlTemplate, "${itemId}", itemInfo.Id))
		q := u.Query()
		q.Set("template_id", itemInfo.MsInfo.TemplateId)
		q.Set(QueryApiKeyName, itemInfo.ApiKey)
		q.Set("openlist_path", itemInfo.MsInfo.OpenlistPath)
		u.RawQuery = q.Encode()
		logs.Success("重定向 playlist: %s", u.String())
		c.Redirect(http.StatusTemporaryRedirect, u.String())
		return
	}

	// 3 请求资源在 Emby 中的 Path 参数
	embyPath, err := getEmbyFileLocalPath(itemInfo)
	if checkErr(c, err) {
		return
	}

	// logs.Info("检查 %s 是否nfs协议的strm文件", embyPath)
	strmUrl := ""
	// nfs协议开头，说明是一个nfs文件路径，打开该路径读取strm内容，然后跳转到strm内的地址
	if strings.HasPrefix(embyPath, "nfs:") && strings.HasSuffix(embyPath, ".strm") {
		logs.Info("检查到 nfs 协议的 strm文件：%s", embyPath)
		// 打开nfs文件
		f, err := os.Open(embyPath)
		if err == nil {
			// 读取strm内容
			buf := bytes.NewBufferString("")
			io.Copy(buf, f)
			strmUrl = buf.String()
			logs.Success("读取到 strm 文件 %s 的内容: %s", embyPath, strmUrl)
			f.Close()
		}
	}
	if strmUrl == "" {
		strmUrl = embyPath
	}
	isProxyUrl := ""
	// 4 如果是远程地址 (strm) 且不包含qmediasync的本地代理播放链接, 重定向处理
	if urls.IsRemote(strmUrl) || strings.HasPrefix(strmUrl, "http") || strings.HasPrefix(strmUrl, "nfs:") {
		finalPath := getFinalRedirectLink(strmUrl, c.Request.Header.Clone())
		if !strings.Contains(finalPath, "/proxy-115") {
			logs.Success("重定向 strm 到直连地址: %s", finalPath)
			c.Header(cache.HeaderKeyExpired, cache.Duration(time.Minute*10))
			c.Redirect(http.StatusTemporaryRedirect, finalPath)
			return
		} else {
			logs.Warn("重定向 strm 包含qmediasync本地代理播放链接，回源处理（会走NAS流量）: %s", finalPath)
			isProxyUrl = finalPath
		}
	}

	// 5 如果是本地地址, 回源处理
	// 1. 以/开头
	// 2. 以windows盘符开头, 正则匹配
	pattern := `^[A-Za-z]:`
	matchedWin, _ := regexp.MatchString(pattern, embyPath)
	// \\开头是Emby网络共享地址
	if strings.HasPrefix(embyPath, "/") || matchedWin || strings.HasPrefix(embyPath, "\\") || isProxyUrl != "" {
		logs.Info("本地或代理路径: %s, 回源处理", embyPath)
		newUri := strings.Replace(c.Request.RequestURI, "stream", "original", 1)
		newUri = strings.Replace(newUri, "universal", "original", 1)
		c.Redirect(http.StatusTemporaryRedirect, newUri)
		return
	}
	// // 6 请求 openlist 资源
	// fi := openlist.FetchInfo{
	// 	Header:       c.Request.Header.Clone(),
	// 	UseTranscode: useTranscode,
	// 	Format:       msInfo.TemplateId,
	// }
	// openlistPathRes := path.Emby2Openlist(embyPath)

	// allErrors := strings.Builder{}
	// // handleOpenlistResource 根据传递的 path 请求 openlist 资源
	// handleOpenlistResource := func(path string) bool {
	// 	logs.Info("尝试请求 Openlist 资源: %s", path)
	// 	fi.Path = path
	// 	res := openlist.FetchResource(fi)

	// 	if res.Code != http.StatusOK {
	// 		allErrors.WriteString(fmt.Sprintf("请求 Openlist 失败, code: %d, msg: %s, path: %s;", res.Code, res.Msg, path))
	// 		return false
	// 	}

	// 	// 处理直链
	// 	if !fi.UseTranscode {
	// 		res.Data.Url = config.C.Emby.Strm.MapPath(res.Data.Url)
	// 		logs.Success("请求成功, 重定向到: %s", res.Data.Url)
	// 		c.Header(cache.HeaderKeyExpired, cache.Duration(time.Minute*10))
	// 		c.Redirect(http.StatusTemporaryRedirect, res.Data.Url)
	// 		return true
	// 	}

	// 	// 代理转码 m3u
	// 	u, _ := url.Parse(https.ClientRequestHost(c.Request) + "/videos/proxy_playlist")
	// 	q := u.Query()
	// 	q.Set("template_id", itemInfo.MsInfo.TemplateId)
	// 	q.Set(QueryApiKeyName, itemInfo.ApiKey)
	// 	q.Set("openlist_path", openlist.PathEncode(path))
	// 	u.RawQuery = q.Encode()
	// 	c.Redirect(http.StatusTemporaryRedirect, u.String())
	// 	return true
	// }

	// if openlistPathRes.Success && handleOpenlistResource(openlistPathRes.Path) {
	// 	return
	// }
	// paths, err := openlistPathRes.Range()
	// if checkErr(c, err) {
	// 	return
	// }
	// if slices.ContainsFunc(paths, func(path string) bool {
	// 	return handleOpenlistResource(path)
	// }) {
	// 	return
	// }

	checkErr(c, fmt.Errorf("没有兼容的流"))
}

// ProxyOriginalResource 拦截 original 接口
func ProxyOriginalResource(c *gin.Context) {
	// if strings.Contains(strings.ToLower(c.Request.RequestURI), "subtitles") {
	// 	ProxyOrigin(c)
	// 	return
	// }

	// itemInfo, err := resolveItemInfo(c, RouteOriginal)
	// if checkErr(c, err) {
	// 	return
	// }

	// embyPath, err := getEmbyFileLocalPath(itemInfo)
	// if checkErr(c, err) {
	// 	return
	// }

	// 如果是本地媒体, 代理回源
	// 2. 以windows盘符开头, 正则匹配
	// pattern := `^[A-Za-z]:`
	// matchedWin, _ := regexp.MatchString(pattern, embyPath)
	// if strings.HasPrefix(embyPath, "/") || matchedWin || !strings.Contains(embyPath, "/proxy-115") {
	ProxyOrigin(c)
	// }
	// Redirect2OpenlistLink(c)
}

// checkErr 检查 err 是否为空
// 不为空则根据错误处理策略返回响应
//
// 返回 true 表示请求已经被处理
func checkErr(c *gin.Context, err error) bool {
	if err == nil || c == nil {
		return false
	}

	// 异常接口, 不缓存
	c.Header(cache.HeaderKeyExpired, "-1")

	// 采用拒绝策略, 直接返回错误
	if config.C.Emby.ProxyErrorStrategy == config.PeStrategyReject {
		logs.Error("代理接口失败: %v", err)
		c.String(http.StatusInternalServerError, "代理接口失败, 请检查日志")
		return true
	}

	logs.Error("代理接口失败: %v, 回源处理", err)
	ProxyOrigin(c)
	return true
}

// getFinalRedirectLink 尝试对带有重定向的原始链接进行内部请求, 返回最终链接
//
// 请求中途出现任何失败都会返回原始链接
func getFinalRedirectLink(originLink string, header http.Header) string {
	if !strings.Contains(originLink, "smartstrm") && (strings.Contains(originLink, "115/newurl") || strings.Contains(originLink, "115/url")) {
		originLink += "&force=1"
	}
	finalLink, resp, err := https.Get(originLink).Header(header).DoRedirect()
	if err != nil {
		logs.Warn("获取最终重定向链接失败, 原始链接: %s, 错误信息: %v", originLink, err)
		return originLink
	}
	logs.Success("获取最终重定向链接成功, 原始链接: %s, 最终链接: %s", originLink, finalLink)
	defer resp.Body.Close()
	return finalLink
}
