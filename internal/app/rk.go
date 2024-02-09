package app

import (
	"encoding/json"
	"html/template"
	"net/http"
	"sort"
	"strings"

	"github.com/tidwall/buntdb"
)

func fRk(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 如果是GET则返回页面
		ud, out := checkUser(r)
		if out {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
			return
		}
		if !ud.IsLogin {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		tmpl, err := template.New("rk").Funcs(template.FuncMap{
			"getrst": getrst,
		}).Parse(RKHTML)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
		pd := getPageData(ud)
		pd.TaskList = getTaskList()
		pd.UserList = sumRk()
		err = tmpl.Execute(w, pd)
		if err != nil {
			elog.Println(err)
			w.Write([]byte("无法渲染页面"))
			return
		}
	} else {
		// 400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

/*
rk排序方式：
AC越多，名字字典序靠前的排名靠前

思路：
获取每个人的全部AC数量，利用sort包排序（sort.SliceStable，strings.Compare）

结构体：
Name, AC
*/

type NameAC struct {
	Name string
	AC   int
}

func sumRk() []string { // getUserSorted
	ul := getUserList()       // 用户列表
	tl := getTaskList()       // 题目列表
	nal := make([]NameAC, 0)  // “用户-AC数”列表
	for nali, u := range ul { // 遍历用户
		nal = append(nal, NameAC{u, 0}) // 新增元素
		for _, t := range tl {          // 遍历题目
			var v string
			var e error
			// 读取数据
			rdb.View(func(tx *buntdb.Tx) error {
				v, e = tx.Get(t + ":" + u)
				return e
			})
			// 解码数据
			var ts TaskStat
			json.Unmarshal([]byte(v), &ts)
			for _, tp := range ts.Details { // 遍历任务点
				if tp.Stat == "AC" {
					nal[nali].AC += 1 // Add 1
				}
			}
		}
	}
	// 排序
	sort.SliceStable(nal, func(i, j int) bool {
		if nal[i].AC != nal[j].AC {
			return nal[i].AC > nal[j].AC
		}
		return strings.Compare(nal[i].Name, nal[j].Name) == -1
	})
	// 重写ul
	for i := range nal {
		ul[i] = nal[i].Name
	}
	return ul
}

func getrst(name, task string) TaskStat {
	var val string
	var e error
	err := rdb.View(func(tx *buntdb.Tx) error {
		val, e = tx.Get(task + ":" + name)
		return e
	})
	if err != nil {
		if err == buntdb.ErrNotFound {
			return TaskStat{"", false, "Unsubmitted", "unsubmitted", nil}
		}
	}
	var ts TaskStat
	json.Unmarshal([]byte(val), &ts)
	return ts
}
