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
	http.HandleFunc("/", fIndex)            // 根页面，用户主页面
	http.HandleFunc("/reg", fReg)           // 注册页面
	http.HandleFunc("/login", fLogin)       // 登录页面
	http.HandleFunc("/exit", fExit)         // 退出登录
	http.HandleFunc("/getSend", fGetSend)   // 下载下发的文件
	http.HandleFunc("/delSend", fDelSend)   // 删除下发的文件
	http.HandleFunc("/upldSend", fUpldSend) // 上传要下发的文件
	// http.HandleFunc("/commit", fCommit)     // 用户提交
	http.HandleFunc("/timer", fTimer) // 计时器
	// http.HandleFunc("/rk", fRk) // 排行榜
	elog.Fatalln(http.ListenAndServe(cfg.Port, nil))
}
