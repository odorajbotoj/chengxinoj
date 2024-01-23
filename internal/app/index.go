package app

import (
	"html/template"
	"net/http"
)

func fIndex(w http.ResponseWriter, r *http.Request) {
	// 如果是GET则返回页面
	_, isl, _ := checkUser(r)
	if !isl {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	tmpl, err := template.New("index").Parse(INDEXHTML)
	if err != nil {
		elog.Println(err)
		w.Write([]byte("无法渲染页面"))
		return
	}
	err = tmpl.Execute(w, getPageData(r))
	if err != nil {
		elog.Println(err)
		w.Write([]byte("无法渲染页面"))
		return
	}
	return
}
