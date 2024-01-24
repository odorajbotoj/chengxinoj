package app

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/tidwall/buntdb"
)

func fReg(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 如果是GET则返回页面
		ud, _ := checkUser(r)
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
		err = tmpl.Execute(w, getPageData(r, ud))
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		return
	} else if r.Method == "POST" {
		// 如果是POST则注册用户
		ud, _ := checkUser(r)
		gvm.RLock()
		cr := canReg
		gvm.RUnlock()
		if !cr && !ud.IsAdmin {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，当前禁止注册");window.location.replace("/");</script>`))
			return
		}

		// 检查表单是否为空
		r.ParseForm()
		if len(r.Form["userRegName"]) == 0 || len(r.Form["userRegMd5"]) == 0 {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，表单为空");window.location.replace("/reg");</script>`))
			return
		}
		// 过滤非法注册admin
		if r.Form["userRegName"][0] == "admin" {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，非法用户名");window.location.replace("/reg");</script>`))
			return
		}
		// 数据操作
		var err error
		err = udb.View(func(tx *buntdb.Tx) error {
			// 过滤重复注册
			_, e := tx.Get("user:" + r.Form["userRegName"][0] + ":passwdMd5")
			return e
		})
		if err != nil {
			if err == buntdb.ErrNotFound {
				// 说明可以注册
				/*
					// 新建用户Data文件夹
					ex, err := exists("recv/" + r.Form["userRegName"][0])
					if err != nil {
						w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，内部发生错误");window.location.replace("/");</script>`))
						elog.Println(err)
						return
					}
					if !ex {
						err = os.Mkdir("recv/"+r.Form["userRegName"][0], 0755)
						if err != nil {
							w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，内部发生错误");window.location.replace("/");</script>`))
							elog.Println(err)
							return
						}
					}
				*/
				// 写入密码散列
				// 写入用户名（方便索引）
				err = udb.Update(func(tx *buntdb.Tx) error {
					_, _, e := tx.Set("user:"+r.Form["userRegName"][0]+":passwdMd5", r.Form["userRegMd5"][0], nil)
					if e != nil {
						return e
					}
					_, _, e = tx.Set("user:"+r.Form["userRegName"][0]+":name", r.Form["userRegName"][0], nil)
					if e != nil {
						return e
					}
					return nil
				})
				if err != nil {
					w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，内部发生错误");window.location.replace("/reg");</script>`))
					elog.Println(err)
					return
				}
				// success
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册成功");window.location.replace("/");</script>`))
				return
			} else {
				// 其他错误
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，内部发生错误");window.location.replace("/reg");</script>`))
				elog.Println(err)
				return
			}
		} else {
			// 用户已存在
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("注册失败，用户已存在");window.location.replace("/reg");</script>`))
			return
		}
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
		err = tmpl.Execute(w, getPageData(r, ud))
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		return
	} else if r.Method == "POST" {
		r.ParseForm()
		if len(r.Form["loginName"]) == 0 || len(r.Form["loginMd5"]) == 0 {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("登录失败，表单为空");window.location.replace("/login");</script>`))
			return
		}
		// 数据操作
		var err error
		var passwdMd5 string
		err = udb.View(func(tx *buntdb.Tx) error {
			var e error
			passwdMd5, e = tx.Get("user:" + r.Form["loginName"][0] + ":passwdMd5")
			return e
		})
		if err != nil {
			if err == buntdb.ErrNotFound {
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("登录失败，用户不存在");window.location.replace("/login");</script>`))
				return
			} else {
				// 其他错误
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("登录失败，内部发生错误");window.location.replace("/login");</script>`))
				elog.Println(err)
				return
			}
		} else {
			// 若用户存在
			if r.Form["loginMd5"][0] != passwdMd5 {
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("密码错误");window.location.replace("/login");</script>`))
				return
			} else {
				sum := md5.Sum([]byte(r.Form["loginName"][0] + "_" + r.Form["loginMd5"][0] + "_" + getIP(r)))
				token := hex.EncodeToString(sum[:])
				err = udb.Update(func(tx *buntdb.Tx) error {
					_, _, e := tx.Set("user:"+r.Form["loginName"][0]+":token", token, nil)
					return e
				})
				if err != nil {
					w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("登录失败，内部发生错误");window.location.replace("/login");</script>`))
					elog.Println(err)
					return
				}
				// set cookies
				c1 := http.Cookie{
					Name:     "username",
					Value:    url.QueryEscape(r.Form["loginName"][0]),
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
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("欢迎 ` + r.Form["loginName"][0] + `");window.location.replace("/");</script>`))
				return
			}
		}
	} else {
		// 400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

func fExit(w http.ResponseWriter, r *http.Request) {
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
}

func checkUser(r *http.Request) (UserData, bool) {
	var ud UserData // 用户信息
	ud.Name = ""
	ud.IsLogin = false
	ud.IsAdmin = false
	c1, err := r.Cookie("username")
	if err != nil {
		if err != http.ErrNoCookie {
			elog.Println(err)
		}
		return ud, false
	}
	ud.Name, err = url.QueryUnescape(c1.Value)
	if err != nil {
		elog.Println(err)
		return ud, false
	}
	c2, err := r.Cookie("token")
	if err != nil {
		if err != http.ErrNoCookie {
			elog.Println(err)
		}
		return ud, false
	}
	var t string = ""
	err = udb.View(func(tx *buntdb.Tx) error {
		t, e = tx.Get("user:" + ud.Name + ":token")
		return e
	})
	if err != nil {
		if err != buntdb.ErrNotFound {
			elog.Println(err)
		}
		return ud, false
	}
	if c2.Value == t {
		ud.IsLogin = true
		if ud.Name == "admin" {
			ud.IsAdmin = true
		}
		return ud, false
	} else {
		return ud, true
	}
}

func getUserList() []string {
	s := make([]string, 0)
	err := udb.View(func(tx *buntdb.Tx) error {
		e := tx.Ascend("name", func(key, value string) bool {
			s = append(s, value)
			return true // continue iteration
		})
		return e
	})
	if err != nil {
		elog.Println(err)
	}
	return s
}

func fListUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 如果是GET则返回页面
		ud, out := checkUser(r)
		if out {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
			return
		}
		if ud.IsLogin && !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		tmpl, err := template.New("listUser").Parse(USERLISTHTML)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		err = tmpl.Execute(w, getPageData(r, ud))
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

func fDelUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		ud, out := checkUser(r)
		if out {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
			return
		}
		if ud.IsLogin && !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		// 接收删除列表
		r.ParseForm()
		lst := r.Form["uname"]
		if len(lst) == 0 {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("删除失败：表单为空");window.location.replace("/listUser");</script>`))
			return
		}
		err := udb.Update(func(tx *buntdb.Tx) error {
			s := make([]string, 0)
			e := tx.Ascend("", func(key, value string) bool {
				ss := strings.Split(key, ":")
				if len(ss) != 3 {
					return true
				}
				if in(ss[1], lst) {
					s = append(s, key)
				}
				return true // continue iteration
			})
			if e != nil {
				return e
			}
			for _, v := range s {
				_, e = tx.Delete(v)
				if e != nil {
					return e
				}
			}
			return nil
		})
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("删除失败：内部错误");window.location.replace("/listUser");</script>`))
			return
		}
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("删除成功");window.location.replace("/listUser");</script>`))
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

// 导入用户数据
func fImpUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		ud, out := checkUser(r)
		if out {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
			return
		}
		if !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024+1024)
		if err := r.ParseMultipartForm(100*1024*1024 + 1024); err != nil {
			http.Error(w, "文件过大，大于100MB", http.StatusBadRequest)
			return
		}
		files, ok := r.MultipartForm.File["file"]
		if !ok || len(files) != 1 { // 出错则取消
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：内部错误（可能提交了空的表单）");window.location.replace("/listUser");</script>`))
			return
		}
		// 建临时库
		tmpdb, err := buntdb.Open(":memory:")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：内部错误");window.location.replace("/listUser");</script>`))
			elog.Println(err)
			return
		}
		defer tmpdb.Close()
		fr, _ := files[0].Open()
		err = tmpdb.Load(fr)
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：内部错误");window.location.replace("/listUser");</script>`))
			elog.Println(err)
			return
		}
		// kv映射
		var kv = make(map[string]string)
		// 读出数据
		err = tmpdb.View(func(tx *buntdb.Tx) error {
			e := tx.Ascend("", func(key, value string) bool {
				kv[key] = value
				return true // continue iteration
			})
			return e
		})
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：内部错误");window.location.replace("/listUser");</script>`))
			return
		}
		// 放进data库
		err = udb.Update(func(tx *buntdb.Tx) error {
			for k, v := range kv {
				_, _, e := tx.Set(k, v, nil)
				if err != nil {
					return e
				}
			}
			return nil
		})
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：内部错误");window.location.replace("/listUser");</script>`))
			return
		}
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入成功");window.location.replace("/listUser");</script>`))
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

// 导出用户数据
func fExpUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		ud, out := checkUser(r)
		if out {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
			return
		}
		if !ud.IsLogin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		gvm.RLock()
		iss := isStarted
		gvm.RUnlock()
		if !iss && !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		// 建立临时库
		tmpdb, err := buntdb.Open(":memory:")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导出失败：内部错误");window.location.replace("/listUser");</script>`))
			elog.Println(err)
			return
		}
		defer tmpdb.Close()
		// 接收导出列表
		r.ParseForm()
		lst := r.Form["uname"]
		if len(lst) == 0 {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导出失败：表单为空");window.location.replace("/listUser");</script>`))
			return
		}
		// kv映射
		var kv = make(map[string]string)
		// 读出数据
		err = udb.View(func(tx *buntdb.Tx) error {
			e := tx.Ascend("", func(key, value string) bool {
				ss := strings.Split(key, ":")
				if len(ss) != 3 {
					return true
				}
				if in(ss[1], lst) {
					kv[key] = value
				}
				return true // continue iteration
			})
			return e
		})
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导出失败：内部错误");window.location.replace("/listUser");</script>`))
			return
		}
		// 放进临时库
		err = tmpdb.Update(func(tx *buntdb.Tx) error {
			for k, v := range kv {
				_, _, e := tx.Set(k, v, nil)
				if err != nil {
					return e
				}
			}
			return nil
		})
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导出失败：内部错误");window.location.replace("/listUser");</script>`))
			return
		}
		// 导出
		var buf bytes.Buffer
		err = tmpdb.Save(&buf)
		if err != nil {
			elog.Println("fExpUser: ", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename=UserList.db")
		w.Header().Set("Content-Type", http.DetectContentType(buf.Bytes()))
		w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
		w.Write(buf.Bytes())
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
