package service

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
)

// 获取教师下发的文件列表
func getSend() map[string]int64 {
	var ret = make(map[string]int64)
	rd, err := os.ReadDir(cfg.SendFileDir)
	if err != nil {
		log.Println("readSend: ", err)
		return ret
	}
	for _, fi := range rd {
		if !fi.IsDir() {
			info, err := fi.Info()
			if err != nil {
				continue
			}
			ret[info.Name()] = info.Size()
		}
	}
	return ret
}

// 下载教师下发的文件
func fSend(w http.ResponseWriter, r *http.Request) {
	_, isLogin, _ := checkUser(w, r)
	if !isLogin {
		return
	}
	if !isStarted {
		return
	}
	fn, err := url.QueryUnescape(r.URL.Query().Get("fn"))
	if fn == "" && err == nil {
		http.Error(w, "404. File not found.", http.StatusNotFound)
	} else if err != nil {
		log.Println("fSend: ", err)
		http.Error(w, err.Error(), http.StatusNotFound)
	} else {
		b, err := os.ReadFile(cfg.SendFileDir + fn)
		if err != nil {
			log.Println("fSend: ", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		finfo, _ := os.Stat(cfg.SendFileDir + fn)
		w.Header().Set("Content-Disposition", "attachment; filename="+fn)
		w.Header().Set("Content-Type", http.DetectContentType(b))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", finfo.Size()))
		w.Write(b)
		return
	}
}
