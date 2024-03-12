package app

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/tidwall/buntdb"
)

type User struct {
	Name  string
	Md5   string
	Token string
}

func fReg(w http.ResponseWriter, r *http.Request) {
	ud, _ := checkUser(r)
	if ud.IsAdmin {
		http.Redirect(w, r, "/regAdmin", http.StatusSeeOther)
		return
	}
	if r.Method == "GET" {
		// 如果是GET则返回页面
		if ud.IsLogin && !ud.IsAdmin {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		tmpl, err := template.New("reg").Parse(USERREGHTML)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		err = tmpl.Execute(w, getPageData(ud))
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		return
	} else if r.Method == "POST" {
		// 如果是POST则注册用户
		gvm.RLock()
		if !canReg && !ud.IsAdmin {
			alertAndRedir(w, "注册失败，当前禁止注册", "/")
			gvm.RUnlock()
			return
		}
		gvm.RUnlock()
		// 检查表单是否为空
		r.ParseForm()
		if len(r.Form["userRegName"]) == 0 || len(r.Form["userRegMd5"]) == 0 {
			alertAndRedir(w, "注册失败，表单为空", "/reg")
			return
		}
		err := userReg(r.Form["userRegName"][0], r.Form["userRegMd5"][0])
		if err != nil {
			alertAndRedir(w, "注册失败："+err.Error(), "/reg")
			elog.Println(err)
			return
		} else {
			alertAndRedir(w, "注册成功", "/")
			log.Printf("用户 %s 注册\n", r.Form["userRegName"][0])
			return
		}
	} else {
		// 400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

func fRegAdmin(w http.ResponseWriter, r *http.Request) {
	ud, out := checkUser(r)
	if out {
		alertAndRedir(w, "请重新登录", "/exit")
		return
	}
	if !ud.IsLogin {
		alertAndRedir(w, "请先登录", "/login")
		return
	}
	if !ud.IsAdmin {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}
	if r.Method == "GET" {
		// 如果是GET则返回页面
		tmpl, err := template.New("regAdmin").Parse(USERREGADMINHTML)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		err = tmpl.Execute(w, getPageData(ud))
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		return
	} else if r.Method == "POST" {
		// 如果是POST则注册用户
		// 检查表单是否为空
		r.ParseForm()
		if len(r.Form["userRegName"]) == 0 || len(r.Form["userRegMd5"]) == 0 {
			alertAndRedir(w, "注册失败，表单为空", "/regAdmin")
			return
		}
		// 分割
		users := splitLine.Split(r.Form.Get("userRegName"), -1)
		if len(users) == 0 {
			alertAndRedir(w, "注册失败，表单为空", "/regAdmin")
			return
		}
		var logstr string = ""
		for _, v := range users {
			if v == "" {
				continue
			}
			err := userReg(v, r.Form["userRegMd5"][0])
			if err != nil {
				logstr += "（批量）注册 " + v + " 失败：" + err.Error() + "\n"
				elog.Println("（批量）注册 " + v + " 失败：" + err.Error())
			} else {
				logstr += "（批量）用户 " + v + " 注册成功\n"
				log.Printf("（批量）用户 %s 注册成功\n", v)
			}
		}
		alertAndRedir(w, logstr, "/regAdmin")
		return
	} else {
		// 400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

func fLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 如果是GET则返回页面
		ud, _ := checkUser(r)
		if ud.IsLogin {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		tmpl, err := template.New("login").Parse(LOGINHTML)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		err = tmpl.Execute(w, getPageData(ud))
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		return
	} else if r.Method == "POST" {
		r.ParseForm()
		if len(r.Form["loginName"]) == 0 || len(r.Form["loginMd5"]) == 0 {
			alertAndRedir(w, "登录失败，表单为空", "/login")
			return
		}
		// 数据操作
		u, err := getUser(r.Form["loginName"][0])
		if err != nil {
			if err == buntdb.ErrNotFound {
				alertAndRedir(w, "登录失败，用户不存在", "/login")
				return
			} else {
				// 其他错误
				alertAndRedir(w, "登录失败："+err.Error(), "/login")
				elog.Println(err)
				return
			}
		}
		// 若用户存在
		if r.Form["loginMd5"][0] != u.Md5 {
			alertAndRedir(w, "密码错误", "/login")
			return
		} else {
			sum := md5.Sum([]byte(u.Name + "_" + u.Md5 + "_" + getIP(r)))
			token := hex.EncodeToString(sum[:])
			u.Token = token
			err = udb.Update(func(tx *buntdb.Tx) error {
				b, e := json.Marshal(u)
				if e != nil {
					return e
				}
				_, _, e = tx.Set("user:"+u.Name+":info", string(b), nil)
				return e
			})
			if err != nil {
				alertAndRedir(w, "登录失败："+err.Error(), "/login")
				elog.Println(err)
				return
			}
			// set cookies
			c1 := http.Cookie{
				Name:     "username",
				Value:    url.QueryEscape(u.Name),
				MaxAge:   16200,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			}
			c2 := http.Cookie{
				Name:     "token",
				Value:    token,
				MaxAge:   16200,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			}
			http.SetCookie(w, &c1)
			http.SetCookie(w, &c2)
			alertAndRedir(w, "欢迎 "+u.Name, "/")
			log.Printf("用户 %s 登录\n", u.Name)
			return
		}
	} else {
		// 400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

func fExit(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("username")
	if err != nil {
		if err != http.ErrNoCookie {
			elog.Println(err)
		}
		return
	}
	uname, err := url.QueryUnescape(c.Value)
	if err != nil {
		elog.Println(err)
		return
	}
	c1 := http.Cookie{
		Name:     "username",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	c2 := http.Cookie{
		Name:     "token",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, &c1)
	http.SetCookie(w, &c2)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	log.Printf("用户 %s 退出登录\n", uname)
}

func fChangePasswd(w http.ResponseWriter, r *http.Request) {
	ud, out := checkUser(r)
	if out {
		alertAndRedir(w, "请重新登录", "/exit")
		return
	}
	if !ud.IsLogin {
		alertAndRedir(w, "请先登录", "/login")
		return
	}
	if ud.IsAdmin {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}
	if r.Method == "GET" {
		// 如果是get就返回页面
		tmpl, err := template.New("changePasswd").Parse(CHANGEPASSWDHTML)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		err = tmpl.Execute(w, getPageData(ud))
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		return
	} else if r.Method == "POST" {
		// 获取用户数据
		u, err := getUser(ud.Name)
		if err != nil {
			if err == buntdb.ErrNotFound {
				alertAndRedir(w, "修改失败，用户不存在", "/changePasswd")
				return
			} else {
				// 其他错误
				alertAndRedir(w, "修改失败："+err.Error(), "/changePasswd")
				elog.Println(err)
				return
			}
		}
		// 比对旧密码正确性
		r.ParseForm()
		old := r.Form.Get("oldPasswdMd5")
		if old != u.Md5 {
			alertAndRedir(w, "修改失败：密码不正确", "/changePasswd")
			return
		}
		// 将旧密码替换为新密码
		newMd5 := r.Form.Get("newPasswdMd5")
		if newMd5 != "" {
			setUser(ud.Name, newMd5)
			alertAndRedir(w, "修改成功", "/")
			log.Printf("用户 %s 修改密码\n", ud.Name)
		} else {
			alertAndRedir(w, "修改失败：新密码为空", "/changePasswd")
		}
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
