package app

import (
	"io"
	"net/http"
	"os"
	"strconv"
)

// 下载下发的文件
func fGetSend(w http.ResponseWriter, r *http.Request) {
	_, isLogin, isAdmin := checkUser(r)
	if !isLogin {
		return
	}
	gvm.RLock()
	if !isStarted && !isAdmin {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}
	gvm.RUnlock()
	fn := r.URL.Query().Get("fn")
	b, err := os.ReadFile("send/" + fn)
	if err != nil {
		elog.Println("fGetSend: ", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	finfo, _ := os.Stat("send/" + fn)
	w.Header().Set("Content-Disposition", "attachment; filename="+fn)
	w.Header().Set("Content-Type", http.DetectContentType(b))
	w.Header().Set("Content-Length", strconv.FormatInt(finfo.Size(), 10))
	w.Write(b)
	return
}

// 删除下发的文件
func fDelSend(w http.ResponseWriter, r *http.Request) {
	_, _, isAdmin := checkUser(r)
	if !isAdmin {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}
	fn := r.URL.Query().Get("fn")
	if fn != "" {
		err := os.Remove("send/" + fn)
		if err != nil {
			elog.Println("fDelSend: ", err)
		}
	}
	http.Redirect(w, r, "/", http.StatusFound)
	return
}

// 上传要下发的文件
func fUpldSend(w http.ResponseWriter, r *http.Request) {
	_, _, isAdmin := checkUser(r)
	if !isAdmin {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024+1024)
	if err := r.ParseMultipartForm(100*1024*1024 + 1024); err != nil {
		http.Error(w, "文件过大，大于100MB", http.StatusBadRequest)
		return
	}
	files, ok := r.MultipartForm.File["file"]
	if !ok { // 出错则取消
		http.Error(w, "未知错误，请重试", http.StatusBadRequest)
		return
	}
	for _, f := range files {
		fr, _ := f.Open()
		fo, err := os.Create("send/" + f.Filename)
		if err != nil {
			elog.Println("fUpldSend: ", err)
			continue
		}
		defer fr.Close()
		defer fo.Close()
		io.Copy(fo, fr)
	}
	http.Redirect(w, r, "/", http.StatusFound)
	return
}
