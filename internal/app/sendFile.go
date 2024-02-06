package app

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

// 下载下发的文件
func fGetSend(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		ud, out := checkUser(r)
		if out {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
			return
		}
		if !ud.IsLogin {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请先登录");window.location.replace("/login");</script>`))
			return
		}
		gvm.RLock()
		iss := isStarted
		gvm.RUnlock()
		if !iss && !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
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
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

// 删除下发的文件
func fDelSend(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		ud, out := checkUser(r)
		if out {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
			return
		}
		if !ud.IsLogin {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请先登录");window.location.replace("/login");</script>`))
			return
		}
		gvm.RLock()
		iss := isStarted
		gvm.RUnlock()
		if iss || !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		r.ParseForm()
		fns := r.Form["fname"]
		if len(fns) == 0 {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("删除失败：表单为空");window.location.replace("/");</script>`))
			return
		}
		for _, fn := range fns {
			err := os.Remove("send/" + fn)
			if err != nil {
				elog.Println("fDelSend: ", err)
			} else {
				log.Println("删除文件：" + fn)
			}
		}
		http.Redirect(w, r, "/", http.StatusFound)
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

// 上传要下发的文件
func fUpldSend(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		ud, out := checkUser(r)
		if out {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
			return
		}
		if !ud.IsLogin {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请先登录");window.location.replace("/login");</script>`))
			return
		}
		gvm.RLock()
		iss := isStarted
		gvm.RUnlock()
		if iss || !ud.IsAdmin {
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
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：内部错误（可能提交了空的表单）");window.location.replace("/");</script>`))
			return
		}
		for _, f := range files {
			fr, _ := f.Open()
			fo, err := os.Create("send/" + f.Filename)
			if err != nil {
				elog.Println("fUpldSend: ", err)
				continue
			} else {
				log.Println("上传文件：" + f.Filename)
			}
			defer fr.Close()
			defer fo.Close()
			io.Copy(fo, fr)
		}
		http.Redirect(w, r, "/", http.StatusFound)
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
