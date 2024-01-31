package app

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/tidwall/buntdb"
)

func fSubmit(w http.ResponseWriter, r *http.Request) {
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
		if !iss && !ud.IsAdmin {
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
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败：内部错误（可能提交了空的表单）");window.location.replace("/task?tn=` + na + `");</script>`))
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
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败：内部错误");window.location.replace("/task?tn=` + na + `");</script>`))
			return
		}
		if handler.Size > taskinfo.MaxSize {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败：文件大小超限");window.location.replace("/task?tn=` + na + `");</script>`))
			return
		}
		// 这里已经拿到了file和taskinfo
		// 用户提交的数据按照用户名分类存放在 recv/ 下
		// 先检查用户目录有没有创建
		err = checkDir("recv/" + ud.Name + "/")
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败：内部错误");window.location.replace("/task?tn=` + na + `");</script>`))
			elog.Println(err)
			return
		}
		// 然后看要不要子文件夹
		pre := "recv/" + ud.Name + "/"
		if taskinfo.SubDir {
			pre = pre + taskinfo.RelName + "/"
		}
		// 检查文件名
		if taskinfo.FileName != handler.Filename {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败：文件名不匹配");window.location.replace("/task?tn=` + na + `");</script>`))
			return
		}
		// 保存文件
		f, err := os.OpenFile(pre+handler.Filename, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交失败：内部错误");window.location.replace("/task?tn=` + na + `");</script>`))
			elog.Println(err)
			return
		}
		io.Copy(f, file)
		f.Close()
		// 看要不要judge
		if taskinfo.Judge {
			log.Println("judge", taskinfo.Title)
			// 接入worker
		}
		w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("提交成功");window.location.replace("/task?tn=` + na + `");</script>`))
		return
	} else {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
