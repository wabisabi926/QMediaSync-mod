package controllers

import (
	"net"
	"net/url"
	"strings"

	"Q115-STRM/internal/helpers"

	"github.com/gin-gonic/gin"
)

var defaultTrustedOrigins = []string{
	"http://localhost:5173",
	"http://127.0.0.1:5173",
	"http://[::1]:5173",
}

func normalizeOrigin(rawOrigin string) (string, bool) {
	rawOrigin = strings.TrimSpace(rawOrigin)
	if rawOrigin == "" {
		return "", false
	}
	parsed, err := url.Parse(rawOrigin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	return normalizeOriginParts(parsed.Scheme, parsed.Host)
}

func normalizeOriginParts(rawScheme string, rawHost string) (string, bool) {
	scheme := strings.ToLower(strings.TrimSpace(rawScheme))
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	parsed, err := url.Parse(scheme + "://" + strings.TrimSpace(rawHost))
	if err != nil || parsed.Host == "" || parsed.Hostname() == "" {
		return "", false
	}
	hostname := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		return scheme + "://" + net.JoinHostPort(hostname, port), true
	}
	if strings.Contains(hostname, ":") {
		hostname = "[" + hostname + "]"
	}
	return scheme + "://" + hostname, true
}

func requestScheme(c *gin.Context) string {
	if forwardedProto := c.Request.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		proto := strings.ToLower(strings.TrimSpace(strings.Split(forwardedProto, ",")[0]))
		if proto == "http" || proto == "https" {
			return proto
		}
	}
	if c.Request.TLS != nil {
		return "https"
	}
	return "http"
}

func currentRequestOrigin(c *gin.Context) string {
	if c.Request.Host == "" {
		return ""
	}
	origin, ok := normalizeOriginParts(requestScheme(c), c.Request.Host)
	if !ok {
		return ""
	}
	return origin
}

func originMatchesList(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		normalizedAllowed, ok := normalizeOrigin(allowed)
		if ok && normalizedAllowed == origin {
			return true
		}
	}
	return false
}

func originAllowed(c *gin.Context, rawOrigin string) bool {
	origin, ok := normalizeOrigin(rawOrigin)
	if !ok {
		return false
	}
	if origin == currentRequestOrigin(c) {
		return true
	}
	if originMatchesList(origin, defaultTrustedOrigins) {
		return true
	}
	return originMatchesList(origin, helpers.GlobalConfig.TrustedOrigins)
}

func requestOriginAllowed(c *gin.Context) bool {
	origin := c.Request.Header.Get("Origin")
	if origin == "" {
		origin = c.Request.Header.Get("Referer")
	}
	if origin == "" {
		return false
	}
	return originAllowed(c, origin)
}
