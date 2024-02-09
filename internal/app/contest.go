package app

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/tidwall/buntdb"
)

// 导入比赛
func fImpContest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		ud, out := checkUser(r)
		if out {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
			return
		}
		if !ud.IsLogin {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请先登录");window.location.replace("/login");</script>`))
			return
		}
		gvm.RLock()
		iss := isStarted
		gvm.RUnlock()
		if iss || !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*1024+1024)
		if err := r.ParseMultipartForm(1024*1024*1024 + 1024); err != nil {
			http.Error(w, "文件过大，大于1GB", http.StatusBadRequest)
			return
		}
		zipf, zipfh, err := r.FormFile("file")
		if err != nil { // 出错则取消
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println(err)
			return
		}
		// 删除目录
		err = os.RemoveAll("send/")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println(err)
			return
		}
		lm := getFileList("tasks/")
		for k := range lm {
			if k != "task.db" && k != "recv.db" {
				err = os.RemoveAll("tasks/" + k)
				if err != nil {
					w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：` + err.Error() + `");window.location.replace("/");</script>`))
					elog.Println(err)
					return
				}
			}
		}
		// 重新加载目录（解压缩）
		err = unzipFile(zipf, zipfh.Size, "./")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println(err)
			return
		}
		// 更新数据库
		// 建临时库
		tmpdb, err := buntdb.Open(":memory:")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println(err)
			return
		}
		defer tmpdb.Close()
		fr, _ := os.Open("task.db")
		err = tmpdb.Load(fr)
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：` + err.Error() + `");window.location.replace("/");</script>`))
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
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：` + err.Error() + `");window.location.replace("/");</script>`))
			return
		}
		// 放进tdb库
		err = tdb.Update(func(tx *buntdb.Tx) error {
			tx.DeleteAll()
			for k, v := range kv {
				_, _, e := tx.Set(k, v, nil)
				if err != nil {
					return e
				}
			}
			return nil
		})
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入失败：` + err.Error() + `");window.location.replace("/");</script>`))
			return
		}
		// 删除拎出来的数据库文件
		err = os.Remove("task.db")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导出失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println("fPackDown: ", err)
			return
		}
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导入成功");window.location.replace("/");</script>`))
		log.Println("导入比赛")
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

// 导出比赛
func fExpContest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		ud, out := checkUser(r)
		if out {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("请重新登录");window.location.replace("/exit");</script>`))
			return
		}
		gvm.RLock()
		iss := isStarted
		gvm.RUnlock()
		if iss || !ud.IsAdmin {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		// 复制数据库
		err := copyFile("tasks/task.db", "task.db")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导出失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println("fPackDown: ", err)
			return
		}
		// 准备压缩数据
		var b = new(bytes.Buffer)
		// 不压缩数据库，所以要把测试点文件单独拎出来
		lm := getFileList("tasks/")
		var l []string
		for k := range lm {
			if k != "task.db" && k != "recv.db" {
				l = append(l, "tasks/"+k)
			}
		}
		l = append(l, "send/", "task.db")
		err = zipFile(b, l...)
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导出失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println("fPackDown: ", err)
			return
		}
		// http导出
		w.Header().Set("Content-Disposition", "attachment; filename=contest.zip")
		w.Header().Set("Content-Type", http.DetectContentType(b.Bytes()))
		w.Header().Set("Content-Length", strconv.Itoa(b.Len()))
		w.Write(b.Bytes())
		// 删除拎出来的数据库文件
		err = os.Remove("task.db")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("导出失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println("fPackDown: ", err)
			return
		}
		log.Println("导出比赛")
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
