package app

import (
	"net/http"

	"github.com/tidwall/buntdb"
)

func Run() {
	// 数据库
	var err error
	db, err = buntdb.Open("db/data.db")
	if err != nil {
		elog.Fatalln(err)
	}
	defer db.Close()
	db.Update(func(tx *buntdb.Tx) error {
		tx.Set("user:admin:passwdMd5", cfg.AdminPasswdMD5, nil)
		return nil
	})

	// 服务器建立

	// 静态资源
	http.Handle("/static/", http.FileServer(http.FS(scriptsFs)))

	// 启动计时器
	go timer()

	// 注册路由
	http.HandleFunc("/", fIndex)
	http.HandleFunc("/reg", fReg)
	http.HandleFunc("/login", fLogin)
	http.HandleFunc("/exit", fExit)
	// http.HandleFunc("/send", fSend)
	// http.HandleFunc("/del", fDel)
	// http.HandleFunc("/upld", fUpld)
	http.HandleFunc("/timer", fTimer)
	// http.HandleFunc("/rk", fRk)
	elog.Fatalln(http.ListenAndServe(cfg.Port, nil))
}
