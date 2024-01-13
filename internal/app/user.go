package service

import (
	"crypto/md5"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
)

func fReg(w http.ResponseWriter, r *http.Request) {
	//
}

func fLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" { // 如果是GET请求就直接重定向走
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	r.ParseForm()
	if len(r.Form["loginName"]) == 0 || len(r.Form["loginMd5"]) == 0 {
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("登录失败，表单为空");window.location.replace("/");</script>`))
		return
	}
	if e, _ := exists(cfg.UserDataDir + r.Form["loginName"][0]); !e {
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("登录失败，用户不存在");window.location.replace("/");</script>`))
		return
	} else {
		bPwd, err := os.ReadFile(cfg.UserDataDir + r.Form["loginName"][0] + "/pass.md5")
		if err != nil {
			log.Println("fLogin: ", err)
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("登录失败，内部发生错误");window.location.replace("/");</script>`))
			return
		}
		if r.Form["loginMd5"][0] != string(bPwd) {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("密码错误");window.location.replace("/");</script>`))
			return
		}
	}
	// set LastLogin
	os.WriteFile(cfg.UserDataDir+r.Form["loginName"][0]+"/lastlogin.txt", []byte(getIP(r)), 0644)
	// set cookie
	c1 := http.Cookie{
		Name:     "username",
		Value:    url.QueryEscape(r.Form["loginName"][0]),
		MaxAge:   16200,
		HttpOnly: true,
	}
	c2 := http.Cookie{
		Name:     "token",
		Value:    fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s%s", r.Form["loginName"][0], r.Form["loginMd5"][0])))),
		MaxAge:   16200,
		HttpOnly: true,
	}
	http.SetCookie(w, &c1)
	http.SetCookie(w, &c2)
	w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("欢迎 ` + r.Form["loginName"][0] + `");window.location.replace("/");</script>`))
}

func fExit(w http.ResponseWriter, r *http.Request) {
	c1 := http.Cookie{
		Name:     "username",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
	}
	c2 := http.Cookie{
		Name:     "token",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, &c1)
	http.SetCookie(w, &c2)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func checkUser(w http.ResponseWriter, r *http.Request) (string, bool, bool) {
	var (
		username string = ""
		isLogin  bool   = false
		isAdmin  bool   = false
	)
	c1, err := r.Cookie("username")
	if err != nil {
		return username, isLogin, isAdmin
	}
	username, err = url.QueryUnescape(c1.Value)
	if err != nil {
		return username, isLogin, isAdmin
	}
	bLastLogin, err := os.ReadFile(cfg.UserDataDir + username + "/lastlogin.txt")
	if err != nil {
		return username, isLogin, isAdmin
	}
	if string(bLastLogin) == getIP(r) {
		c2, err := r.Cookie("token")
		if err == nil {
			bPwd, err := os.ReadFile(cfg.UserDataDir + username + "/pass.md5")
			if err == nil && c2.Value == fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s%s", username, string(bPwd))))) {
				isLogin = true
				if username == "admin" {
					isAdmin = true
				}
			}
		}
	} else {
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("该用户已从其他IP登录");window.location.replace("/exit");</script>`))
	}
	return username, isLogin, isAdmin
}
