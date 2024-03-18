package app

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/tidwall/buntdb"
)

type TaskPoint struct {
	Name         string // 名字
	Introduction string // 简介
	SubDir       bool   // 建立子文件夹
	MaxSize      int64  // 最大文件大小（字节）
	FileType     string // 允许的后缀

	Judge    bool   // 是否评测（以下内容仅在此选项为真时有意义）
	FileIO   bool   // 文件输入输出（否则是标准输入输出）
	CC       string // 编译器
	CFlags   string // 编译选项
	Duration int    // 时限（毫秒）
}

type TaskStat struct {
	Md5     string            // 校验和
	Judge   bool              // 是否评测（以下内容仅在此选项为真时有意义）
	Stat    string            // 评测状态
	Info    string            // 输出的信息
	Details map[int]TestPoint // 测试点状态
}

type TestPoint struct {
	Stat string // 结果
	Info string // 详情
	Time int64  // 耗时（毫秒）
}

// 任务点页面
func fTask(w http.ResponseWriter, r *http.Request) {
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
		gvm.RLock()
		if !isStarted && !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			gvm.RUnlock()
			return
		}
		gvm.RUnlock()
		var tn = r.URL.Query().Get("tn")
		var task TaskPoint
		var s string
		err := tdb.View(func(tx *buntdb.Tx) error {
			var e error
			s, e = tx.Get("task:" + tn + ":info")
			return e
		})
		if err != nil && err == buntdb.ErrNotFound {
			http.Error(w, "404 Not Found", http.StatusNotFound)
			return
		}
		err = json.Unmarshal([]byte(s), &task)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			elog.Println(err)
			return
		}
		pd := getPageData(ud)
		pd.Task = task
		tmpl, err := template.New("task").Parse(TASKHTML)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
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

// 新建任务点
func fNewTask(w http.ResponseWriter, r *http.Request) {
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
		gvm.RLock()
		if isStarted || !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			gvm.RUnlock()
			return
		}
		gvm.RUnlock()
		var ntn = r.URL.Query().Get("ntname")
		if ntn == "" {
			alertAndRedir(w, "新建失败，表单为空", "/")
			return
		}
		err := tdb.View(func(tx *buntdb.Tx) error {
			_, e := tx.Get("task:" + ntn + ":info")
			return e
		})
		if (err == nil) || (err != buntdb.ErrNotFound) {
			alertAndRedir(w, "新建失败，存在同名任务点", "/")
			return
		}
		err = tdb.Update(func(tx *buntdb.Tx) error {
			var info TaskPoint
			info.Name = ntn
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
			alertAndRedir(w, "新建失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		alertAndRedir(w, "新建成功", "/editTask?tn="+ntn)
		log.Println("新建任务：", ntn)
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

// 编辑任务点
func fEditTask(w http.ResponseWriter, r *http.Request) {
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
		gvm.RLock()
		if isStarted || !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			gvm.RUnlock()
			return
		}
		gvm.RUnlock()
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
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		pd.Task = t
		err = tmpl.Execute(w, pd)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		return
	} else if r.Method == "POST" {
		// 如果是POST就处理表单
		r.ParseForm()
		var t TaskPoint
		var err error
		t.Name = r.Form.Get("tn")
		// 继续填充内容
		t.Introduction = r.Form.Get("introduction")
		if r.Form.Get("subd") == "subd" {
			t.SubDir = true
		}
		var ms int64
		ms, err = strconv.ParseInt(r.Form.Get("maxs"), 10, 64)
		if err != nil {
			alertAndRedir(w, "保存失败："+err.Error(), "/editTask?tn="+t.Name)
			elog.Println(err)
			return
		}
		t.MaxSize = ms
		t.FileType = r.Form.Get("fileType")
		if r.Form.Get("recvOrJudge") == "judge" {
			t.Judge = true
			if r.Form.Get("fileOrStd") == "fileIO" {
				t.FileIO = true
			}
			t.CC = r.Form.Get("cc")
			isCCE, err := exists(t.CC)
			if !isCCE || err != nil {
				alertAndRedir(w, "保存失败：提交的编译器不存在", "/editTask?tn="+t.Name)
				return
			}
			t.CFlags = r.Form.Get("cflags")
			var d int
			d, err = strconv.Atoi(r.Form.Get("duration"))
			if err != nil {
				alertAndRedir(w, "保存失败："+err.Error(), "/editTask?tn="+t.Name)
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
			_, _, e = tx.Set("task:"+t.Name+":info", string(b), nil)
			return e
		})
		if err != nil {
			alertAndRedir(w, "保存失败："+err.Error(), "/editTask?tn="+t.Name)
			elog.Println(err)
			return
		}
		alertAndRedir(w, "保存成功", "/editTask?tn="+t.Name)
		log.Println("保存任务信息 " + t.Name)
		if t.Judge {
			// 如果是能测试的任务点就重测
			go reJudgeTask(t)
		}
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

// 删除任务点
func fDelTask(w http.ResponseWriter, r *http.Request) {
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
		gvm.RLock()
		if isStarted || !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			gvm.RUnlock()
			return
		}
		gvm.RUnlock()
		// 接收删除列表
		r.ParseForm()
		lst := r.Form["tname"]
		if len(lst) == 0 {
			alertAndRedir(w, "删除失败：表单为空", "/")
			return
		}
		var err error
		for _, v := range lst {
			err = os.RemoveAll("tasks/" + v + "/")
			if err != nil {
				alertAndRedir(w, "删除失败："+err.Error(), "/")
				elog.Println(err)
				return
			}
			log.Println("删除测试数据：" + v)
		}
		// 在tdb里面删除
		err = tdb.Update(func(tx *buntdb.Tx) error {
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
			alertAndRedir(w, "删除失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		// 在rdb里面删除
		err = rdb.Update(func(tx *buntdb.Tx) error {
			s := make([]string, 0) // 待删除名单
			e := tx.Ascend("", func(key, value string) bool {
				ss := strings.Split(key, ":")
				if len(ss) != 2 {
					return true
				}
				if in(ss[0], lst) {
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
			alertAndRedir(w, "删除失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		alertAndRedir(w, "删除成功", "/")
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

// 获取任务列表
func getTaskList() []string {
	s := make([]string, 0)
	err := tdb.View(func(tx *buntdb.Tx) error {
		e := tx.Ascend("taskInfo", func(key, value string) bool {
			var info TaskPoint
			json.Unmarshal([]byte(value), &info)
			s = append(s, info.Name)
			return true // continue iteration
		})
		return e
	})
	if err != nil && err != buntdb.ErrNotFound {
		elog.Println(err)
	}
	return s
}

// 上传测试数据
func fUpldTest(w http.ResponseWriter, r *http.Request) {
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
		gvm.RLock()
		if isStarted || !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			gvm.RUnlock()
			return
		}
		gvm.RUnlock()
		r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024+1024)
		if err := r.ParseMultipartForm(100*1024*1024 + 1024); err != nil {
			http.Error(w, "文件过大，大于100MB", http.StatusBadRequest)
			return
		}
		zipf, zipfh, err := r.FormFile("testpoints")
		na := r.Form.Get("tn")
		if err != nil { // 出错则取消
			alertAndRedir(w, "上传失败："+err.Error(), "/editTask?tn="+na)
			elog.Println(err)
			return
		}
		err = checkDir("tasks/" + na + "/")
		if err != nil {
			alertAndRedir(w, "上传失败："+err.Error(), "/editTask?tn="+na)
			elog.Println(err)
			return
		}
		err = unzipFile(zipf, zipfh.Size, "tasks/"+na+"/")
		if err != nil {
			alertAndRedir(w, "上传失败："+err.Error(), "/editTask?tn="+na)
			elog.Println(err)
			return
		}
		var t TaskPoint
		err = tdb.View(func(tx *buntdb.Tx) error {
			s, e := tx.Get("task:" + na + ":info")
			if e != nil {
				return e
			}
			e = json.Unmarshal([]byte(s), &t)
			return e
		})
		if err != nil {
			alertAndRedir(w, "上传失败："+err.Error(), "/editTask?tn="+na)
			elog.Println(err)
			return
		}
		go reJudgeTask(t)
		alertAndRedir(w, "上传成功", "/editTask?tn="+na)
		log.Println("上传测试点 " + na)
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

// 删除测试数据
func fDelTest(w http.ResponseWriter, r *http.Request) {
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
		gvm.RLock()
		if isStarted || !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			gvm.RUnlock()
			return
		}
		gvm.RUnlock()
		r.ParseForm() // 别忘了，否则拿到的是空的
		na := r.Form.Get("tn")
		if na == "" {
			alertAndRedir(w, "清空失败，参数为空", "/")
			return
		}
		err := os.RemoveAll("tasks/" + na + "/")
		if err != nil {
			alertAndRedir(w, "清空失败："+err.Error(), "/editTask?tn="+na)
			elog.Println(err)
			return
		}
		alertAndRedir(w, "清空成功", "/editTask?tn="+na)
		log.Println("清空上传的测试点 " + na)
		return
	} else {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
