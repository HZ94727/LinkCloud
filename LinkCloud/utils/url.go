package utils

import "net/url"

// IsValidURL 验证 URL 格式是否正确
func IsValidURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// 必须有协议（http/https）
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	// 必须有域名
	if parsed.Host == "" {
		return false
	}

	return true
}
