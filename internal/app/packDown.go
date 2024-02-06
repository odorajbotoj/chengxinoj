package app

import (
	"bytes"
	"net/http"
	"strconv"
)

func fPackDown(w http.ResponseWriter, r *http.Request) {
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
		err := zipFile(b, "recv/")
		if err != nil {
			elog.Println("fPackDown: ", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename=recv.zip")
		w.Header().Set("Content-Type", http.DetectContentType(b.Bytes()))
		w.Header().Set("Content-Length", strconv.Itoa(b.Len()))
		w.Write(b.Bytes())
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
