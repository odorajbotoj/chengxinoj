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
		select {
		case t := <-timerStart:
			log.Println("比赛开始")
			gvm.Lock()
			isStarted = true
			startTime = time.Now().UnixMilli()
			duration = t * 60
			gvm.Unlock()
			if t <= 0 {
				// 阻塞直到取消
				select {
				case <-timerEnd:
					gvm.Lock()
					isStarted = false
					startTime = 0
					duration = 0
					gvm.Unlock()
					log.Println("比赛结束")
				case <-stopSignal:
					log.Println("计时器已停止")
					wg.Done()
					return
				}
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
				case <-stopSignal:
					log.Println("计时器已停止")
					wg.Done()
					return
				}
			}
		case <-stopSignal:
			log.Println("计时器已停止")
			wg.Done()
			return
		}
	}
}

func fTimer(w http.ResponseWriter, r *http.Request) {
	ud, out := checkUser(r)
	if out {
		alertAndRedir(w, "请重新登录", "/exit")
		return
	}
	if ud.Name == "admin" && ud.IsLogin && ud.IsAdmin {
		gvm.RLock()
		defer gvm.RUnlock()
		if r.Method == "POST" && !isStarted { // 开始比赛
			r.ParseForm()
			dl := r.Form.Get("durationLimit")
			if dl == "on" { // 限制时间
				td := r.Form.Get("timeDuration")
				t, err := strconv.ParseInt(td, 10, 64)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				} else if t > 300 {
					http.Error(w, "400 Bad Request", http.StatusBadRequest)
					return
				}
				timerStart <- t
			} else { // 不限时间
				timerStart <- -1
			}
		} else if r.Method == "GET" && isStarted { // 结束比赛
			timerEnd <- 0
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}
}
