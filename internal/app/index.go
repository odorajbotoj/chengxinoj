package app

import (
	"html/template"
	"net/http"
)

func fIndex(w http.ResponseWriter, r *http.Request) {
	// 如果是GET则返回页面
	ud, out := checkUser(r)
	if out {
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
		return
	}
	if !ud.IsLogin {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	tmpl, err := template.New("index").Parse(INDEXHTML)
	if err != nil {
		elog.Println(err)
		w.Write([]byte("无法渲染页面"))
		return
	}
	err = tmpl.Execute(w, getPageData(r, ud))
	if err != nil {
		elog.Println(err)
		w.Write([]byte("无法渲染页面"))
		return
	}
	return
}
