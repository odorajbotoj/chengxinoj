package app

import (
	"embed"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
)

const (
	// 项目基本信息
	VERSION string = "v0.0.3" // 版本号
)

var (
	// 全局变量读写锁
	gvm sync.RWMutex
	// 是否开始比赛
	isStarted bool = false
	// 是否允许注册
	canReg bool = false
	// 比赛开始时间
	startTime int64 = 0
	// 比赛延续时间
	duration int64 = 0
)

// 加载网页资源
//
//go:embed static/html/base.html
var BASEHTML string

//go:embed static/html/index.html
var INDEXHTML string

//go:embed static/html/userReg.html
var USERREGHTML string

//go:embed static/html/login.html
var LOGINHTML string

//go:embed static/html/user.html
var USERLISTHTML string

//go:embed static/html/editTask.html
var EDITTASKHTML string

//go:embed static/html/task.html
var TASKHTML string

//go:embed static/html/rk.html
var RKHTML string

//go:embed static/scripts
var scriptsFs embed.FS

// 全局设置
var cfg Config

// 专用输出错误信息
var elog *log.Logger

// 正则
var goodUserName *regexp.Regexp // 合法用户名
var splitLine *regexp.Regexp    // 分行

func Init() {
	// 新建elog, 专用输出错误信息
	eLogFile, err := os.OpenFile("log.txt", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln("Init: cannot create log file: ", err)
	}
	elog = log.New(io.MultiWriter(eLogFile, os.Stdout), "[ERR] ", log.Ldate|log.Lmicroseconds|log.Lshortfile)

	// 编译正则
	goodUserName = regexp.MustCompile("^[\u4E00-\u9FA5A-Za-z0-9_]{2,20}$")
	splitLine = regexp.MustCompile(`[\t\n\f\r]`)

	// 检查配置文件
	err = checkConfig()
	if err != nil {
		elog.Fatalln("Init: checkConfig: ", err)
	}
	// 读取配置文件
	err = readConfigTo(&cfg)
	if err != nil {
		elog.Fatalln("Init: readConfigTo: ", err)
	}

	// 检查文件夹是否存在，不存在则创建
	err = checkDir("userdb/")
	if err != nil {
		elog.Fatalln("Init: checkDir: ", err)
	}
	err = checkDir("recvFiles/")
	if err != nil {
		elog.Fatalln("Init: checkDir: ", err)
	}
	err = checkDir("send/")
	if err != nil {
		elog.Fatalln("Init: checkDir: ", err)
	}
	err = checkDir("test/")
	if err != nil {
		elog.Fatalln("Init: checkDir: ", err)
	}
	err = checkDir("tasks/")
	if err != nil {
		elog.Fatalln("Init: checkDir: ", err)
	}

	// 加载网页模板
	INDEXHTML = strings.Replace(BASEHTML, "<!--REPLACE-->", INDEXHTML, 1)
	USERREGHTML = strings.Replace(BASEHTML, "<!--REPLACE-->", USERREGHTML, 1)
	LOGINHTML = strings.Replace(BASEHTML, "<!--REPLACE-->", LOGINHTML, 1)
	USERLISTHTML = strings.Replace(BASEHTML, "<!--REPLACE-->", USERLISTHTML, 1)
	EDITTASKHTML = strings.Replace(BASEHTML, "<!--REPLACE-->", EDITTASKHTML, 1)
	TASKHTML = strings.Replace(BASEHTML, "<!--REPLACE-->", TASKHTML, 1)
	RKHTML = strings.Replace(BASEHTML, "<!--REPLACE-->", RKHTML, 1)
}
