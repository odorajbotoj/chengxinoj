package app

import (
	"log"
	"net/http"
	"strconv"
	"time"
)

var (
	timerStart = make(chan int64)
	timerEnd   = make(chan int64)
)

func timer() {
	for {
		t := <-timerStart
		log.Println("比赛开始")
		gvm.Lock()
		isStarted = true
		startTime = time.Now().UnixMilli()
		duration = t * 60
		gvm.Unlock()
		if t <= 0 {
			// 阻塞直到取消
			<-timerEnd
			gvm.Lock()
			isStarted = false
			startTime = 0
			duration = 0
			gvm.Unlock()
			log.Println("比赛结束")
		} else {
			select {
			// 等待取消或超时
			case <-timerEnd:
				gvm.Lock()
				isStarted = false
				startTime = 0
				duration = 0
				gvm.Unlock()
				log.Println("比赛结束")
			case <-time.After(time.Duration(t) * time.Minute):
				gvm.Lock()
				isStarted = false
				startTime = 0
				duration = 0
				gvm.Unlock()
				log.Println("比赛结束")
			}
		}
	}
}

func fTimer(w http.ResponseWriter, r *http.Request) {
	name, isLogin, isAdmin := checkUser(r)
	if name == "admin" && isLogin && isAdmin {
		if r.Method == "GET" && !isStarted {
			dl := r.URL.Query().Get("durationLimit")
			td := r.URL.Query().Get("timeDuration")
			if dl == "on" {
				t, err := strconv.ParseInt(td, 10, 64)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				timerStart <- t
			} else {
				timerStart <- -1
			}
		} else if r.Method == "POST" && isStarted {
			timerEnd <- 0
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusBadRequest)
	}
}
