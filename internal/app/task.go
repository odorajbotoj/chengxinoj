package app

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/tidwall/buntdb"
)

type TaskPoint struct {
	Title        string // 标题
	RelName      string // 真实名字
	Introduction string // 简介
	AcceptFile   string // 允许的文件类型
	SubDir       bool   // 建立子文件夹

	Judge     bool   // 是否评测（以下内容仅在此选项为真时有意义）
	FileIO    bool   // 文件输入输出（否则是标准输入输出）
	ShowScore bool   // 显示分数（OI赛制）（否则是ACM赛制）
	CC        string // 编译器
	CFlags    string // 编译选项
	Duration  int64  // 时限（毫秒）
}

func fImpContest(w http.ResponseWriter, r *http.Request) {
	return
}

func fExpContest(w http.ResponseWriter, r *http.Request) {
	return
}

func fTask(w http.ResponseWriter, r *http.Request) {
	return
}

func fEditTask(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 如果是GET则返回页面
		ud, out := checkUser(r)
		if out {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
			return
		}
		if !ud.IsLogin {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请先登录");window.location.replace("/login");</script>`))
			return
		}
		if !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		tmpl, err := template.New("editTask").Parse(EDITTASKHTML)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		var t TaskPoint
		pd := getPageData(ud)
		var tn = r.URL.Query().Get("tn")
		err = tdb.View(func(tx *buntdb.Tx) error {
			s, e := tx.Get("task:" + tn + ":info")
			if e != nil {
				return e
			}
			e = json.Unmarshal([]byte(s), &t)
			return e
		})
		pd.Task = t
		err = tmpl.Execute(w, pd)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		return
	} else if r.Method == "POST" {
		r.ParseForm()
		var t TaskPoint
		var err error
		t.Title = r.Form.Get("title")
		t.RelName = r.Form.Get("relName")
		t.Introduction = r.Form.Get("introduction")
		t.AcceptFile = r.Form.Get("acf")
		if r.Form.Get("subd") == "subd" {
			t.SubDir = true
		}
		if r.Form.Get("recvOrJudge") == "judge" {
			t.Judge = true
			if r.Form.Get("fileOrStd") == "fileIO" {
				t.FileIO = true
			}
			if r.Form.Get("shows") == "shows" {
				t.ShowScore = true
			}
			t.CC = r.Form.Get("cc")
			t.CFlags = r.Form.Get("cflags")
			var d int64
			d, err = strconv.ParseInt(r.Form.Get("duration"), 10, 64)
			if err != nil {
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("保存失败，内部发生错误");window.location.replace("/editTask?tn=` + t.Title + `");</script>`))
				elog.Println(err)
				return
			}
			t.Duration = d
		}
		err = tdb.Update(func(tx *buntdb.Tx) error {
			b, e := json.Marshal(t)
			if e != nil {
				return e
			}
			_, _, e = tx.Set("task:"+t.Title+":info", string(b), nil)
			return e
		})
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("保存失败，内部发生错误");window.location.replace("/editTask?tn=` + t.Title + `");;</script>`))
			elog.Println(err)
			return
		}
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("保存成功");window.location.replace("/editTask?tn=` + t.Title + `");</script>`))
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

func fNewTask(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 如果是GET则返回页面
		ud, out := checkUser(r)
		if out {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
			return
		}
		if !ud.IsLogin {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请先登录");window.location.replace("/login");</script>`))
			return
		}
		if !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		var ntn = r.URL.Query().Get("ntname")
		if ntn == "" {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("新建失败，表单为空");window.location.replace("/");</script>`))
			return
		}
		err := tdb.Update(func(tx *buntdb.Tx) error {
			var info TaskPoint
			info.Title = ntn
			info.FileIO = true
			info.Judge = false
			b, e := json.Marshal(info)
			if e != nil {
				return e
			}
			_, _, e = tx.Set("task:"+ntn+":info", string(b), nil)
			if e != nil {
				return e
			}
			return nil
		})
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("新建失败，内部发生错误");window.location.replace("/");</script>`))
			elog.Println(err)
			return
		}
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("新建成功");window.location.replace("/");</script>`))
		log.Println("新建任务：", ntn)
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

func fDelTask(w http.ResponseWriter, r *http.Request) {
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
		// 接收删除列表
		r.ParseForm()
		lst := r.Form["tname"]
		if len(lst) == 0 {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("删除失败：表单为空");window.location.replace("/");</script>`))
			return
		}
		err := tdb.Update(func(tx *buntdb.Tx) error {
			s := make([]string, 0) // 待删除名单
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
				} else {
					log.Println("删除：", v)
				}
			}
			return nil
		})
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("删除失败：内部错误");window.location.replace("/");</script>`))
			return
		}
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("删除成功");window.location.replace("/");</script>`))
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

func getTaskList() []string {
	s := make([]string, 0)
	err := tdb.View(func(tx *buntdb.Tx) error {
		e := tx.Ascend("taskInfo", func(key, value string) bool {
			var info TaskPoint
			json.Unmarshal([]byte(value), &info)
			s = append(s, info.Title)
			return true // continue iteration
		})
		return e
	})
	if err != nil && err != buntdb.ErrNotFound {
		elog.Println(err)
	}
	return s
}
