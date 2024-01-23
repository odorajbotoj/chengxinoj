package main

import (
	"chengxinoj/internal/app"
	"log"
)

func init() {
	// 输出基本信息
	log.Println("Chengxin OJ")
	log.Println(app.VERSION)
	log.Println("odorajbotoj(xuezihao)")

	// 加载配置文件
	log.Println("Init...")
	app.Init()
	log.Println("Init Done.")
}

func main() {
	// 主程序入口
	app.Run()
}
