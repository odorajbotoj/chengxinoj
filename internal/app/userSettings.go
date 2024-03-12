package app

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/tidwall/buntdb"
)

func fListUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 如果是GET则返回页面
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
		tmpl, err := template.New("listUser").Parse(USERLISTHTML)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		pd := getPageData(ud)
		pd.UserList = getUserList()
		err = tmpl.Execute(w, pd)
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

// 删除用户
func fDelUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
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
		// 接收删除列表
		r.ParseForm()
		lst := r.Form["uname"]
		if len(lst) == 0 {
			alertAndRedir(w, "删除失败：表单为空", "/listUser")
			return
		}
		// 在udb里面删除
		err := udb.Update(func(tx *buntdb.Tx) error {
			s := make([]string, 0) // 待删除的key名单
			e := tx.Ascend("", func(key, value string) bool {
				ss := strings.Split(key, ":")
				if len(ss) != 3 {
					return true
				}
				if ss[1] != "admin" && in(ss[1], lst) {
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
				} else {
					log.Println("删除：" + v)
				}
			}
			return nil
		})
		if err != nil {
			elog.Println(err)
			alertAndRedir(w, "删除失败："+err.Error(), "/listUser")
			return
		}
		// 在rdb里面删除
		err = rdb.Update(func(tx *buntdb.Tx) error {
			s := make([]string, 0) // 待删除的key名单
			e := tx.Ascend("", func(key, value string) bool {
				ss := strings.Split(key, ":")
				if len(ss) != 2 {
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
				} else {
					log.Println("删除：" + v)
				}
			}
			return nil
		})
		if err != nil {
			elog.Println(err)
			alertAndRedir(w, "删除失败："+err.Error(), "/listUser")
			return
		}
		// 删除用户目录
		for _, v := range lst {
			err = os.RemoveAll("recvFiles/" + v)
			if err != nil {
				elog.Println(err)
			} else {
				log.Println("删除：recvFiles/" + v)
			}
		}
		alertAndRedir(w, "删除成功", "/listUser")
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
		r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024+1024)
		if err := r.ParseMultipartForm(100*1024*1024 + 1024); err != nil {
			http.Error(w, "文件过大，大于100MB", http.StatusBadRequest)
			return
		}
		files, ok := r.MultipartForm.File["file"]
		if !ok || len(files) != 1 { // 出错则取消
			alertAndRedir(w, "导入失败：内部错误（可能提交了空的表单）", "/listUser")
			return
		}
		// 建临时库
		tmpdb, err := buntdb.Open(":memory:")
		if err != nil {
			alertAndRedir(w, "导入失败："+err.Error(), "/listUser")
			elog.Println(err)
			return
		}
		defer tmpdb.Close()
		fr, _ := files[0].Open()
		fr.Close()
		err = tmpdb.Load(fr)
		if err != nil {
			alertAndRedir(w, "导入失败："+err.Error(), "/listUser")
			elog.Println(err)
			return
		}
		// kv映射
		var kv = make(map[string]string)
		// 读出数据
		err = tmpdb.View(func(tx *buntdb.Tx) error {
			e := tx.Ascend("", func(key, value string) bool {
				ss := strings.Split(key, ":")
				if len(ss) != 3 {
					return true
				}
				if ss[0] == "user" && ss[2] == "info" {
					kv[key] = value
				}
				return true // continue iteration
			})
			return e
		})
		if err != nil {
			alertAndRedir(w, "导入失败："+err.Error(), "/listUser")
			return
		}
		// 放进data库
		err = udb.Update(func(tx *buntdb.Tx) error {
			for k, v := range kv {
				if k == "user:admin:info" {
					continue
				}
				_, _, e := tx.Set(k, v, nil)
				if err != nil {
					return e
				}
			}
			return nil
		})
		if err != nil {
			alertAndRedir(w, "导入失败："+err.Error(), "/listUser")
			return
		}
		alertAndRedir(w, "导入成功", "/listUser")
		log.Println("导入用户")
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
		// 接收导出列表
		r.ParseForm()
		lst := r.Form["uname"]
		if len(lst) == 0 {
			alertAndRedir(w, "导出失败：表单为空", "/listUser")
			return
		}
		// 建立临时库
		tmpdb, err := buntdb.Open(":memory:")
		if err != nil {
			alertAndRedir(w, "导出失败："+err.Error(), "/listUser")
			elog.Println(err)
			return
		}
		defer tmpdb.Close()
		// kv映射
		var kv = make(map[string]string)
		// 读出数据
		err = udb.View(func(tx *buntdb.Tx) error {
			e := tx.Ascend("name", func(key, value string) bool {
				ss := strings.Split(key, ":")
				if len(ss) != 3 {
					return true
				}
				if ss[1] != "admin" && in(ss[1], lst) {
					kv[key] = value
				}
				return true // continue iteration
			})
			return e
		})
		if err != nil {
			alertAndRedir(w, "导出失败："+err.Error(), "/listUser")
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
			alertAndRedir(w, "导出失败："+err.Error(), "/listUser")
			return
		}
		// 导出
		var buf = new(bytes.Buffer)
		err = tmpdb.Save(buf)
		if err != nil {
			elog.Println("fExpUser: ", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename=UserList.db")
		w.Header().Set("Content-Type", http.DetectContentType(buf.Bytes()))
		w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
		w.Write(buf.Bytes())
		log.Println("导出用户")
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

func fCanReg(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
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
		var action = r.URL.Query().Get("action")
		if action == "on" {
			gvm.Lock()
			canReg = true
			gvm.Unlock()
			log.Println("开启用户注册")
		} else if action == "off" {
			gvm.Lock()
			canReg = false
			gvm.Unlock()
			log.Println("禁止用户注册")
		} else {
			//400
			http.Error(w, "400 Bad Request", http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

func fResetPasswd(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
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
		r.ParseForm()
		lst := r.Form["uname"]
		if len(lst) == 0 {
			alertAndRedir(w, "重设失败：表单为空", "/listUser")
			return
		}
		if r.Form.Get("rstMd5") == "" {
			alertAndRedir(w, "重设失败：密码为空", "/listUser")
			return
		}
		for _, v := range lst {
			err := setUser(v, r.Form.Get("rstMd5"))
			if err != nil {
				alertAndRedir(w, "重设失败："+err.Error(), "/listUser")
				return
			}
		}
		alertAndRedir(w, "重设成功", "/listUser")
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
