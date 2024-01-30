package app

import (
	"log"
	"net/http"
)

func fCanSubmit(w http.ResponseWriter, r *http.Request) {
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
		if !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		var action = r.URL.Query().Get("action")
		if action == "on" {
			gvm.Lock()
			canSubmit = true
			gvm.Unlock()
			log.Println("开放提交")
		} else if action == "off" {
			gvm.Lock()
			canSubmit = false
			gvm.Unlock()
			log.Println("禁止提交")
		} else {
			//400
			http.Error(w, "400 Bad Request", http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
