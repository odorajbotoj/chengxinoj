package app

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"strconv"
)

// 导入比赛
func fImpContest(w http.ResponseWriter, r *http.Request) {
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
		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*1024+1024)
		if err := r.ParseMultipartForm(1024*1024*1024 + 1024); err != nil {
			http.Error(w, "文件过大，大于1GB", http.StatusBadRequest)
			return
		}
		zipf, zipfh, err := r.FormFile("file")
		if err != nil { // 出错则取消
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("上传失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println(err)
			return
		}
		// shanchu
		err = os.RemoveAll("recvFiles/")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("上传失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println(err)
			return
		}
		err = os.RemoveAll("send/")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("上传失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println(err)
			return
		}
		err = os.RemoveAll("tasks/")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("上传失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println(err)
			return
		}
		err = unzipFile(zipf, zipfh.Size, "./")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("上传失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println(err)
			return
		}
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("上传成功");window.location.replace("/");</script>`))
		log.Println("导入比赛")
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

// 导出比赛
func fExpContest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		ud, out := checkUser(r)
		if out {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
			return
		}
		gvm.RLock()
		iss := isStarted
		gvm.RUnlock()
		if iss || !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		var b = new(bytes.Buffer)
		err := zipFile(b, "recvFiles/", "send/", "tasks/")
		if err != nil {
			elog.Println("fPackDown: ", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename=contest.zip")
		w.Header().Set("Content-Type", http.DetectContentType(b.Bytes()))
		w.Header().Set("Content-Length", strconv.Itoa(b.Len()))
		w.Write(b.Bytes())
		log.Println("导出比赛")
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
