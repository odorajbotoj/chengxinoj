package app

import (
	"crypto/md5"
	"encoding/hex"
	"html/template"
	"net/http"
	"net/url"
	"os"

	"github.com/tidwall/buntdb"
)

func fReg(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 如果是GET则返回页面
		tmpl, err := template.New("reg").Parse(USERREGHTML)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		err = tmpl.Execute(w, getPageData(r))
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		return
	} else if r.Method == "POST" {
		// 如果是POST则注册用户
		if !canReg {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，当前禁止注册");window.location.replace("/");</script>`))
			return
		}
		// 检查表单是否为空
		r.ParseForm()
		if len(r.Form["userRegName"]) == 0 || len(r.Form["userRegMd5"]) == 0 {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，表单为空");window.location.replace("/");</script>`))
			return
		}
		// 过滤非法注册admin
		if r.Form["userRegName"][0] == "admin" {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，非法用户名");window.location.replace("/");</script>`))
			return
		}
		// 数据操作
		var err error
		db.View(func(tx *buntdb.Tx) error {
			// 过滤重复注册
			_, err = tx.Get("user:" + r.Form["userRegName"][0] + ":passwdMd5")
			return err
		})
		if err != nil {
			if err == buntdb.ErrNotFound {
				// 说明可以注册
				// 新建用户Data文件夹
				ex, err := exists("recv/" + r.Form["userRegName"][0])
				if err != nil {
					w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，内部发生错误");window.location.replace("/");</script>`))
					elog.Println(err)
					return
				}
				if !ex {
					err = os.Mkdir("recv/"+r.Form["userRegName"][0], 0644)
					if err != nil {
						w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，内部发生错误");window.location.replace("/");</script>`))
						elog.Println(err)
						return
					}
				}
				// 写入密码散列
				db.Update(func(tx *buntdb.Tx) error {
					_, _, err = tx.Set("user:"+r.Form["userRegName"][0]+":passwdMd5", r.Form["userRegMd5"][0], nil)
					return err
				})
				if err != nil {
					w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，内部发生错误");window.location.replace("/");</script>`))
					elog.Println(err)
					return
				}
				// success
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册成功");window.location.replace("/");</script>`))
				return
			} else {
				// 其他错误
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，内部发生错误");window.location.replace("/");</script>`))
				elog.Println(err)
				return
			}
		} else {
			// 用户已存在
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，用户已存在");window.location.replace("/");</script>`))
			return
		}
	} else {
		// 400
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
}

func fLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 如果是GET则返回页面
		tmpl, err := template.New("login").Parse(LOGINHTML)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		err = tmpl.Execute(w, getPageData(r))
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		return
	} else if r.Method == "POST" {
		r.ParseForm()
		if len(r.Form["loginName"]) == 0 || len(r.Form["loginMd5"]) == 0 {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("登录失败，表单为空");window.location.replace("/");</script>`))
			return
		}
		// 数据操作
		var err error
		var passwdMd5 string
		db.View(func(tx *buntdb.Tx) error {
			passwdMd5, err = tx.Get("user:" + r.Form["loginName"][0] + ":passwdMd5")
			return err
		})
		if err != nil {
			if err == buntdb.ErrNotFound {
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("登录失败，用户不存在");window.location.replace("/");</script>`))
				return
			} else {
				// 其他错误
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，内部发生错误");window.location.replace("/");</script>`))
				elog.Println(err)
				return
			}
		} else {
			// 用户存在
			if r.Form["loginMd5"][0] != passwdMd5 {
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("密码错误");window.location.replace("/");</script>`))
				return
			} else {
				// set cookies
				sum := md5.Sum([]byte(r.Form["loginName"][0] + r.Form["loginMd5"][0]))
				c1 := http.Cookie{
					Name:     "username",
					Value:    url.QueryEscape(r.Form["loginName"][0]),
					MaxAge:   16200,
					HttpOnly: true,
				}
				c2 := http.Cookie{
					Name:     "token",
					Value:    hex.EncodeToString(sum[:]),
					MaxAge:   16200,
					HttpOnly: true,
				}
				http.SetCookie(w, &c1)
				http.SetCookie(w, &c2)
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("欢迎 ` + r.Form["loginName"][0] + `");window.location.replace("/");</script>`))
				return
			}
		}
	} else {
		// 400
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
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

func checkUser(r *http.Request) (string, bool, bool) {
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
	c2, err := r.Cookie("token")
	if err == nil {
		err = db.View(func(tx *buntdb.Tx) error {
			bPwd, err := tx.Get("user:" + username + ":passwdMd5")
			sum := md5.Sum([]byte(username + bPwd))
			if err == nil && c2.Value == hex.EncodeToString(sum[:]) {
				isLogin = true
				if username == "admin" {
					isAdmin = true
				}
			}
			return err
		})
		if err != nil {
			elog.Println(err)
		}
	}
	return username, isLogin, isAdmin
}
