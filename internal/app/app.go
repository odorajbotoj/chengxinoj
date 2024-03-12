package app

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"

	"github.com/tidwall/buntdb"
)

// 数据库
var udb *buntdb.DB // user database
var tdb *buntdb.DB // task database
var rdb *buntdb.DB // recv database

// waitgroup
var wg sync.WaitGroup

// 终止信号用channel
var (
	signalListener = make(chan os.Signal)
	stopSignal     = make(chan struct{})
)

// stop信号广播
func sl() {
	<-signalListener
	close(signalListener)
	close(stopSignal)
	log.Println("正在停止服务")
	wg.Done()
}
func Run() {
	// 监听终止信号
	signal.Notify(signalListener, os.Interrupt)
	wg.Add(4)
	go sl()

	// 数据库
	// task database
	var err error
	tdb, err = buntdb.Open("tasks/task.db")
	if err != nil {
		elog.Fatalln(err)
	}
	defer tdb.Close()
	tdb.Shrink()
	tdb.CreateIndex("taskInfo", "task:*:info", buntdb.IndexJSON("Name"))

	// user database
	udb, err = buntdb.Open("userdb/user.db")
	if err != nil {
		elog.Fatalln(err)
	}
	defer udb.Close()
	udb.Shrink()
	err = udb.Update(func(tx *buntdb.Tx) error {
		var admin User
		admin.Name = "admin"
		admin.Md5 = cfg.AdminPasswdMD5
		admin.Token = ""
		b, e := json.Marshal(admin)
		if e != nil {
			return e
		}
		_, _, e = tx.Set("user:admin:info", string(b), nil)
		return e
	})
	if err != nil {
		elog.Fatalln(err)
	}
	udb.CreateIndex("name", "user:*:info", buntdb.IndexJSON("Name"))

	// recv database
	rdb, err = buntdb.Open("tasks/recv.db")
	if err != nil {
		elog.Fatalln(err)
	}
	defer rdb.Close()
	rdb.Shrink()

	// 服务器建立

	// 启动计时器
	go timer()

	// 启动judger
	go judger()

	// 注册路由
	mux := http.NewServeMux()
	mux.Handle("/static/", http.FileServer(http.FS(scriptsFs))) // 静态资源
	mux.HandleFunc("/", fIndex)                                 // 根页面，用户主页面

	mux.HandleFunc("/reg", fReg)                   // 注册页面
	mux.HandleFunc("/login", fLogin)               // 登录页面
	mux.HandleFunc("/changePasswd", fChangePasswd) // 修改密码
	mux.HandleFunc("/exit", fExit)                 // 退出登录

	mux.HandleFunc("/getSend", fGetSend)   // 下载下发的文件
	mux.HandleFunc("/delSend", fDelSend)   // 删除下发的文件
	mux.HandleFunc("/upldSend", fUpldSend) // 上传要下发的文件

	mux.HandleFunc("/timer", fTimer) // 计时器

	mux.HandleFunc("/canReg", fCanReg)           // 注册开关
	mux.HandleFunc("/listUser", fListUser)       // 用户列表
	mux.HandleFunc("/delUser", fDelUser)         // 删除用户
	mux.HandleFunc("/impUser", fImpUser)         // 导入用户
	mux.HandleFunc("/expUser", fExpUser)         // 导出用户
	mux.HandleFunc("/resetPasswd", fResetPasswd) // 重设密码

	mux.HandleFunc("/packDown", fPackDown)   // 打包下载
	mux.HandleFunc("/clearRecv", fClearRecv) // 清空上传

	mux.HandleFunc("/impContest", fImpContest) // 导入比赛
	mux.HandleFunc("/expContest", fExpContest) // 导出比赛

	mux.HandleFunc("/task", fTask)         // 查看任务
	mux.HandleFunc("/editTask", fEditTask) // 编辑任务
	mux.HandleFunc("/newTask", fNewTask)   // 新建任务
	mux.HandleFunc("/delTask", fDelTask)   // 删除任务

	mux.HandleFunc("/upldTest", fUpldTest) // 上传测试点
	mux.HandleFunc("/delTest", fDelTest)   // 删除测试点

	mux.HandleFunc("/submit", fSubmit)           // 用户提交
	mux.HandleFunc("/clearSubmit", fClearSubmit) // 用户清空提交

	mux.HandleFunc("/rk", fRk) // 排行榜

	var srv = new(http.Server)
	srv.Addr = cfg.Port
	srv.Handler = mux
	go func() {
		<-stopSignal
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			elog.Printf("HTTP server Shutdown: %v\n", err)
		}
		log.Println("网页服务已停止")
		wg.Done()
	}()
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		elog.Fatalf("HTTP server ListenAndServe: %v\n", err)
	}
	wg.Wait()
	log.Println("主程序退出")
}
