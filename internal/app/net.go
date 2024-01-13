package service

import (
	"net/http"
	"strings"
)

// 获取ip地址
func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED_FOR")
	if forwarded != "" {
		return strings.Split(forwarded, ":")[0]
	} else {
		return strings.Split(r.RemoteAddr, ":")[0]
	}
}
