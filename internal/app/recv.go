package app

import (
	"crypto/md5"
	"encoding/hex"
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
		gvm.RLock()
		if !isStarted {
			alertAndRedir(w, "提交失败，当前禁止提交", "/")
			gvm.RUnlock()
			return
		}
		gvm.RUnlock()
		// 解析
		r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024+1024)
		if err := r.ParseMultipartForm(100*1024*1024 + 1024); err != nil {
			http.Error(w, "文件过大", http.StatusBadRequest)
			return
		}
		file, handler, err := r.FormFile("submitFile")
		na := r.Form.Get("subFN")
		if err != nil { // 出错则取消
			alertAndRedir(w, "提交失败："+err.Error(), "/task?tn="+na)
			return
		}
		defer file.Close()
		// 取出题目信息
		var taskinfo TaskPoint
		err = tdb.View(func(tx *buntdb.Tx) error {
			s, e := tx.Get("task:" + na + ":info")
			if e != nil {
				return e
			}
			e = json.Unmarshal([]byte(s), &taskinfo)
			return e
		})
		if err != nil {
			alertAndRedir(w, "提交失败："+err.Error(), "/task?tn="+na)
			return
		}
		// 检查文件大小
		if handler.Size > taskinfo.MaxSize {
			alertAndRedir(w, "提交失败：文件大小超限", "/task?tn="+na)
			return
		}
		// 检查文件名
		if taskinfo.Name+taskinfo.FileType != handler.Filename {
			alertAndRedir(w, "提交失败：文件名不匹配", "/task?tn="+na)
			return
		}
		// 这里已经拿到了file和taskinfo
		// 应该先校验是否和之前提交的内容一样
		var ts TaskStat
		err = rdb.View(func(tx *buntdb.Tx) error {
			s, e := tx.Get(taskinfo.Name + ":" + ud.Name)
			if e != nil {
				if e == buntdb.ErrNotFound {
					return nil
				} else {
					return e
				}
			}
			e = json.Unmarshal([]byte(s), &ts)
			return e
		})
		if err != nil {
			alertAndRedir(w, "提交失败："+err.Error(), "/task?tn="+na)
			elog.Println(err)
			return
		}
		md5h := md5.New()
		io.Copy(md5h, file)
		file.Seek(0, 0)
		newMd5 := hex.EncodeToString(md5h.Sum(nil))
		// 如果一致则跳过，不一致则存储
		if newMd5 != ts.Md5 {
			// 用户提交的数据按照用户名分类存放在 recvFiles/ 下
			// 先检查用户目录有没有创建
			err = checkDir("recvFiles/" + ud.Name + "/")
			if err != nil {
				alertAndRedir(w, "提交失败："+err.Error(), "/task?tn="+na)
				elog.Println(err)
				return
			}
			// 然后看要不要子文件夹
			pre := "recvFiles/" + ud.Name + "/"
			if taskinfo.SubDir {
				pre = pre + taskinfo.Name + "/"
				err = checkDir(pre)
				if err != nil {
					alertAndRedir(w, "提交失败："+err.Error(), "/task?tn="+na)
					elog.Println(err)
					return
				}
			}
			// 保存文件
			f, err := os.Create(pre + handler.Filename) // 不能用OpenFile因为它不清空内容
			if err != nil {
				alertAndRedir(w, "提交失败："+err.Error(), "/task?tn="+na)
				elog.Println(err)
				return
			}
			io.Copy(f, file)
			f.Close() // 千万不要盲目defer
			// 写入提交数据库
			err = rdb.Update(func(tx *buntdb.Tx) error {
				var t TaskStat
				t.Md5 = newMd5
				t.Judge = taskinfo.Judge
				if t.Judge {
					t.Stat = "Waiting"
				} else {
					t.Stat = "Submitted"
				}
				b, e := json.Marshal(t)
				if e != nil {
					return e
				}
				_, _, e = tx.Set(taskinfo.Name+":"+ud.Name, string(b), nil)
				return e
			})
			if err != nil {
				alertAndRedir(w, "提交失败："+err.Error(), "/task?tn="+na)
				elog.Println(err)
				return
			}
			// 看要不要judge
			if taskinfo.Judge {
				go func() {
					judgeQueue <- JudgeTask{ud.Name, taskinfo}
				}()
			}
		}
		log.Println("用户 " + ud.Name + " 提交 " + taskinfo.Name)
		alertAndRedir(w, "提交成功", "/task?tn="+na)
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
			alertAndRedir(w, "请重新登录", "/exit")
			return
		}
		if !ud.IsLogin {
			alertAndRedir(w, "请先登录", "/login")
			return
		}
		gvm.RLock()
		if !isStarted && !ud.IsAdmin {
			alertAndRedir(w, "清空失败，当前禁止清空", "/")
			gvm.RUnlock()
			return
		}
		gvm.RUnlock()
		// 删除提交文件
		err := os.RemoveAll("recvFiles/")
		if err != nil {
			alertAndRedir(w, "清空失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		err = os.MkdirAll("recvFiles/", 0755)
		if err != nil {
			alertAndRedir(w, "清空失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		// 删除提交记录
		err = rdb.Update(func(tx *buntdb.Tx) error {
			return tx.DeleteAll()
		})
		if err != nil {
			alertAndRedir(w, "清空失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		alertAndRedir(w, "清空成功", "/")
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
		gvm.RLock()
		if !isStarted {
			alertAndRedir(w, "清空失败，当前禁止清空", "/")
			gvm.RUnlock()
			return
		}
		gvm.RUnlock()
		// 删除提交的文件
		err := os.RemoveAll("recvFiles/" + ud.Name + "/")
		if err != nil {
			alertAndRedir(w, "清空失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		// 删除提交记录
		err = rdb.Update(func(tx *buntdb.Tx) error {
			keys := getTaskList()
			for _, v := range keys {
				_, e := tx.Delete(v + ":" + ud.Name)
				if e != nil {
					if e == buntdb.ErrNotFound {
						continue
					} else {
						return e
					}
				}
			}
			return nil
		})
		if err != nil {
			alertAndRedir(w, "清空失败："+err.Error(), "/")
			elog.Println(err)
			return
		}
		alertAndRedir(w, "清空成功", "/")
		log.Println("用户 " + ud.Name + " 清空个人上传及记录")
		return
	} else {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
