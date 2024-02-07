package app

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/tidwall/buntdb"
)

// 处理用户上传
func fSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" { // 上传
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
		if !iss {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败，当前禁止提交");window.location.replace("/");</script>`))
			return
		}
		// 解析
		r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024+1024)
		if err := r.ParseMultipartForm(100*1024*1024 + 1024); err != nil {
			http.Error(w, "文件过大", http.StatusBadRequest)
			return
		}
		file, handler, err := r.FormFile("submitFile")
		na := r.Form.Get("subFN")
		var taskinfo TaskPoint
		if err != nil { // 出错则取消
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败：` + err.Error() + `");window.location.replace("/task?tn=` + na + `");</script>`))
			return
		}
		defer file.Close()
		err = tdb.View(func(tx *buntdb.Tx) error {
			s, e := tx.Get("task:" + na + ":info")
			if e != nil {
				return e
			}
			e = json.Unmarshal([]byte(s), &taskinfo)
			return e
		})
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败：` + err.Error() + `");window.location.replace("/task?tn=` + na + `");</script>`))
			return
		}
		if handler.Size > taskinfo.MaxSize {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败：文件大小超限");window.location.replace("/task?tn=` + na + `");</script>`))
			return
		}
		// 检查文件名
		if taskinfo.Name+taskinfo.FileType != handler.Filename {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败：文件名不匹配");window.location.replace("/task?tn=` + na + `");</script>`))
			return
		}
		// 这里已经拿到了file和taskinfo
		// 用户提交的数据按照用户名分类存放在 recv/ 下
		// 先检查用户目录有没有创建
		err = checkDir("recv/" + ud.Name + "/")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败：` + err.Error() + `");window.location.replace("/task?tn=` + na + `");</script>`))
			elog.Println(err)
			return
		}
		// 然后看要不要子文件夹
		pre := "recv/" + ud.Name + "/"
		if taskinfo.SubDir {
			pre = pre + taskinfo.Name + "/"
			err = checkDir(pre)
			if err != nil {
				w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败：` + err.Error() + `");window.location.replace("/task?tn=` + na + `");</script>`))
				elog.Println(err)
				return
			}
		}
		// 保存文件
		f, err := os.OpenFile(pre+handler.Filename, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败：` + err.Error() + `");window.location.replace("/task?tn=` + na + `");</script>`))
			elog.Println(err)
			return
		}
		defer f.Close()
		io.Copy(f, file)
		// 写入提交数据库
		// TODO
		// 看要不要judge
		if taskinfo.Judge {
			log.Println("judge", taskinfo.Name)
			// 接入worker
		}
		log.Println("用户 " + ud.Name + " 提交 " + taskinfo.Name)
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交成功");window.location.replace("/task?tn=` + na + `");</script>`))
		return
	} else {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

// 清除全部用户上传
func fClearRecv(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
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
		if !iss && !ud.IsAdmin {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("清空失败，当前禁止清空");window.location.replace("/");</script>`))
			return
		}
		err := os.RemoveAll("recv/")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("清空失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println(err)
			return
		}
		err = os.MkdirAll("recv/", 0755)
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("清空失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println(err)
			return
		}
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("清空成功");window.location.replace("/");</script>`))
		log.Println("清空所有用户上传及记录")
		return
	} else {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}

// 用户清除个人上传
func fClearSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
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
		if !iss {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("清空失败，当前禁止清空");window.location.replace("/");</script>`))
			return
		}
		err := os.RemoveAll("recv/" + ud.Name + "/")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("清空失败：` + err.Error() + `");window.location.replace("/");</script>`))
			elog.Println(err)
			return
		}
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("清空成功");window.location.replace("/");</script>`))
		log.Println("用户 " + ud.Name + " 清空个人上传及记录")
		return
	} else {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
