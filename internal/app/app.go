package service

import (
	"log"
	"net/http"
)

func Run() {
	// 服务器建立

	// 静态资源
	//go:embed static/scripts
	var scriptsFs embed.FS
	http.Handle("/static/", http.FileServer(http.FS(scriptsFs)))

	// 启动计时器
	go timer()

	// 注册路由
	http.HandleFunc("/", fIndex)
	http.HandleFunc("/reg", fReg)
	http.HandleFunc("/login", fLogin)
	http.HandleFunc("/exit", fExit)
	http.HandleFunc("/send", fSend)
	http.HandleFunc("/del", fDel)
	http.HandleFunc("/upld", fUpld)
	http.HandleFunc("/timer", fTimer)
	http.HandleFunc("/rk", fRk)
	elog.Fatalln(http.ListenAndServe(cfg.Port, nil))
}
