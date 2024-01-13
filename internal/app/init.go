package service

import (
	"embed"
	"html/template"
	"io"
	"log"
	"os"
	"sync"
)

const (
	// 项目基本信息
	VERSION string = "v1.0.0" // 版本号
)

var (
	// 全局变量读写锁
	gvm sync.RWMutex
	// 网页模板
	mainTemplate *template.Template
	// 是否开始比赛
	isStarted bool = false
	// 比赛开始时间
	startTime int64 = 0
	// 比赛延续时间
	duration int64 = 0
)

// 全局设置
var cfg Config

func Init() {
	// 新建elog, 专用输出错误信息
	eLogFile, err := os.OpenFile("log.txt", os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Fatalln("Init: cannot create log file: ", err)
	}
	elog := log.New(io.MultiWriter(eLogFile, os.Stdout), "[ERR] ", log.Ldate|log.Lmicroseconds|log.Lshortfile)

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
	err = checkDir("db/")
	if err != nil {
		elog.Fatalln("Init: checkDir: ", err)
	}
	err = checkDir("recv/")
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
	err = checkDir("prob/")
	if err != nil {
		elog.Fatalln("Init: checkDir: ", err)
	}

	// 加载模板
	mainTemplate, err = loadTemplate()
	if err != nil {
		elog.Fatalln("loadTemplate: ", err)
	}
}
