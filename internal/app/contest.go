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
		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*1024+1024)
		if err := r.ParseMultipartForm(1024*1024*1024 + 1024); err != nil {
			http.Error(w, "文件过大，大于1GB", http.StatusBadRequest)
			return
		}
		zipf, zipfh, err := r.FormFile("file")
		if err != nil { // 出错则取消
			alertAndRedir(w, "导入失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		// 检查合法性
		err = checkUpldContestZip(zipf, zipfh.Size)
		if err != nil {
			alertAndRedir(w, "导入失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		// 删除目录
		err = os.RemoveAll("send/")
		if err != nil {
			alertAndRedir(w, "导入失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		// 删除提交文件
		err = os.RemoveAll("recvFiles/")
		if err != nil {
			alertAndRedir(w, "导入失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		err = os.MkdirAll("recvFiles/", 0755)
		if err != nil {
			alertAndRedir(w, "导入失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		// 删除提交记录
		err = rdb.Update(func(tx *buntdb.Tx) error {
			return tx.DeleteAll()
		})
		if err != nil {
			alertAndRedir(w, "导入失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		list := getTaskList()
		for _, v := range list {
			ok, _ := exists("tasks/" + v)
			if ok {
				err = os.RemoveAll("tasks/" + v)
				if err != nil {
					alertAndRedir(w, "导入失败："+err.Error(), "/")
					elog.Println(err)
					return
				}
			}
		}
		// 重新加载目录（解压缩）
		err = unzipFile(zipf, zipfh.Size, "./")
		if err != nil {
			alertAndRedir(w, "导入失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		// 更新数据库
		// 建临时库
		tmpdb, err := buntdb.Open("task.db")
		if err != nil {
			alertAndRedir(w, "导入失败："+err.Error(), "/")
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
			alertAndRedir(w, "导入失败："+err.Error(), "/")
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
			alertAndRedir(w, "导入失败："+err.Error(), "/")
			return
		}
		// 删除拎出来的数据库文件
		tmpdb.Close()
		err = os.Remove("task.db")
		if err != nil {
			alertAndRedir(w, "导入失败："+err.Error(), "/")
			elog.Println("fPackDown: ", err)
			return
		}
		alertAndRedir(w, "导入成功", "/")
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
		// 复制数据库
		err := copyFile("tasks/task.db", "task.db")
		if err != nil {
			alertAndRedir(w, "导出失败："+err.Error(), "/")
			elog.Println("fPackDown: ", err)
			return
		}
		// 准备压缩数据
		var b = new(bytes.Buffer)
		// 不压缩数据库，所以要把测试点文件单独拎出来
		list := getTaskList()
		var zipFList []string
		for _, v := range list {
			ok, _ := exists("tasks/" + v)
			if ok {
				zipFList = append(zipFList, "tasks/"+v)
			}
		}
		zipFList = append(zipFList, "send/", "task.db")
		err = zipFile(b, zipFList...)
		if err != nil {
			alertAndRedir(w, "导出失败："+err.Error(), "/")
			elog.Println("fPackDown: ", err)
			return
		}
		// 删除拎出来的数据库文件
		err = os.Remove("task.db")
		if err != nil {
			alertAndRedir(w, "导出失败："+err.Error(), "/")
			elog.Println("fPackDown: ", err)
			return
		}
		log.Println("导出比赛")
		// http导出
		w.Header().Set("Content-Disposition", "attachment; filename=contest.zip")
		w.Header().Set("Content-Type", http.DetectContentType(b.Bytes()))
		w.Header().Set("Content-Length", strconv.Itoa(b.Len()))
		w.Write(b.Bytes())
		return
	} else {
		//400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
