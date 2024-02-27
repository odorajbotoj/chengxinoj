package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/buntdb"
)

type JudgeTask struct {
	User UserData
	Task TaskPoint
}

var judgeQueue = make(chan JudgeTask)

// 评测函数
func judger() {
	var exe string = "outbin.exe"
	if runtime.GOOS != "windows" {
		exe = "./" + exe
	}
	for {
		select {
		case jt := <-judgeQueue:
			log.Println("评测" + jt.Task.Name + ":" + jt.User.Name)
			// 清空test目录
			err := os.RemoveAll("test/")
			if err != nil {
				elog.Println(err)
				continue
			}
			err = os.MkdirAll("test/", 0755)
			if err != nil {
				elog.Println(err)
				continue
			}
			// 检查有没有测试点
			ext, err := exists("tasks/" + jt.Task.Name)
			if err != nil {
				elog.Println(err)
				sumRst(jt.User.Name, jt.Task.Name, "Inner Error", err.Error(), nil)
				continue
			}
			if !ext {
				elog.Println("评测" + jt.Task.Name + "找不到任务点")
				sumRst(jt.User.Name, jt.Task.Name, "Inner Error", "评测"+jt.Task.Name+"找不到任务点", nil)
				continue
			}
			// 获取任务点个数
			fl := getFileList("tasks/" + jt.Task.Name + "/")
			cnt := len(fl)
			if cnt%2 != 0 {
				elog.Println("评测" + jt.Task.Name + "任务点个数不匹配")
				sumRst(jt.User.Name, jt.Task.Name, "Inner Error", "评测"+jt.Task.Name+"任务点个数不匹配", nil)
				continue
			}
			cnt /= 2
			// 编译
			// 复制文件
			if jt.Task.SubDir {
				err = copyFile("recvFiles/"+jt.User.Name+"/"+jt.Task.Name+"/"+jt.Task.Name+jt.Task.FileType, "test/src"+jt.Task.FileType)
			} else {
				err = copyFile("recvFiles/"+jt.User.Name+"/"+jt.Task.Name+jt.Task.FileType, "test/src"+jt.Task.FileType)
			}
			if err != nil {
				elog.Println(err)
				sumRst(jt.User.Name, jt.Task.Name, "Inner Error", err.Error(), nil)
				continue
			}
			// 执行编译，生成outbin.exe（为了windows/unix通用）
			cf := strings.Split(jt.Task.CFlags+" -o outbin.exe", " ")
			cf = append(cf, "src"+jt.Task.FileType)
			cCmd := exec.Command(jt.Task.CC, cf...)
			cCmd.Dir = "test/"
			out, err := cCmd.CombinedOutput()
			if err != nil {
				sumRst(jt.User.Name, jt.Task.Name, "CE", string(out), nil)
				continue
			}
			// 运行
			// 循环，评测每个点
			var allOK = true
			var m = make(map[string]TestPoint) // 储存每个点的状态
			if jt.Task.FileIO {
				// 文件输入输出
				for i := 1; i <= cnt; i++ {
					// 拷贝输入文件
					err = copyFile(fmt.Sprintf("tasks/%s/%s%d.in", jt.Task.Name, jt.Task.Name, i), "test/"+jt.Task.Name+".in")
					if err != nil {
						if !os.IsNotExist(err) {
							elog.Println(err)
							sumRst(jt.User.Name, jt.Task.Name, "Inner Error", err.Error(), nil)
							allOK = false
						}
						break

					}
					// 执行
					ctx, cancel := context.WithTimeout(context.Background(), time.Duration(jt.Task.Duration)*time.Millisecond) // 设置运行超时
					defer cancel()
					rCmd := exec.CommandContext(ctx, exe) // 创建运行进程实例
					rCmd.Dir = "test/"
					_, err = rCmd.CombinedOutput()
					if ctx.Err() == context.DeadlineExceeded { // 超时 TLE
						m[strconv.Itoa(i)] = TestPoint{"TLE", ctx.Err().Error()}
						continue
					}
					if err != nil { // 运行出错 RE
						m[strconv.Itoa(i)] = TestPoint{"RE", err.Error()}
						continue
					}
					// 比较输出
					ansBytes, err := os.ReadFile(fmt.Sprintf("tasks/%s/%s%d.out", jt.Task.Name, jt.Task.Name, i))
					if err != nil {
						elog.Println(err)
						sumRst(jt.User.Name, jt.Task.Name, "Inner Error", err.Error(), m)
						allOK = false
						break
					}
					outBytes, err := os.ReadFile(fmt.Sprintf("test/%s.out", jt.Task.Name))
					if err != nil {
						elog.Println(err)
						sumRst(jt.User.Name, jt.Task.Name, "Inner Error", err.Error(), m)
						allOK = false
						break
					}
					// 转字符串，去前后空白，按行分割
					ans := string(ansBytes)
					out := string(outBytes)
					ans = strings.TrimSpace(ans)
					out = strings.TrimSpace(out)
					ansLines := splitLine.Split(ans, -1)
					outLines := splitLine.Split(out, -1)
					if len(ansLines) != len(outLines) {
						m[strconv.Itoa(i)] = TestPoint{"WA", "wrong answer"}
						continue
					}
					m[strconv.Itoa(i)] = TestPoint{"AC", "accepted"}
					for j := 0; j < len(ansLines); j++ {
						if ansLines[j] != outLines[j] {
							m[strconv.Itoa(i)] = TestPoint{"WA", "wrong answer"}
							break
						}
					}
				}
			} else {
				// 标准输入输出
				for i := 1; i <= cnt; i++ {
					// 拷贝输入文件
					inpFile, err := os.Open(fmt.Sprintf("tasks/%s/%s%d.in", jt.Task.Name, jt.Task.Name, i))
					if err != nil {
						if !os.IsNotExist(err) {
							elog.Println(err)
							sumRst(jt.User.Name, jt.Task.Name, "Inner Error", err.Error(), nil)
							allOK = false
						}
						break
					}
					// 执行
					ctx, cancel := context.WithTimeout(context.Background(), time.Duration(jt.Task.Duration)*time.Millisecond) // 设置运行超时
					defer cancel()
					rCmd := exec.CommandContext(ctx, exe) // 创建运行进程实例
					rCmd.Dir = "test/"
					rCmd.Stdin = inpFile // 输入重定向
					outBytes, err := rCmd.CombinedOutput()
					if ctx.Err() == context.DeadlineExceeded { // 超时 TLE
						m[strconv.Itoa(i)] = TestPoint{"TLE", ctx.Err().Error()}
						continue
					}
					if err != nil { // 运行出错 RE
						m[strconv.Itoa(i)] = TestPoint{"RE", err.Error()}
						continue
					}
					// 比较输出
					ansBytes, err := os.ReadFile(fmt.Sprintf("tasks/%s/%s%d.out", jt.Task.Name, jt.Task.Name, i))
					if err != nil {
						elog.Println(err)
						sumRst(jt.User.Name, jt.Task.Name, "Inner Error", err.Error(), m)
						allOK = false
						break
					}
					// 转字符串，去前后空白，按行分割
					ans := string(ansBytes)
					out := string(outBytes)
					ans = strings.TrimSpace(ans)
					out = strings.TrimSpace(out)
					ansLines := splitLine.Split(ans, -1)
					outLines := splitLine.Split(out, -1)
					if len(ansLines) != len(outLines) {
						m[strconv.Itoa(i)] = TestPoint{"WA", "wrong answer"}
						continue
					}
					m[strconv.Itoa(i)] = TestPoint{"AC", "accepted"}
					for j := 0; j < len(ansLines); j++ {
						if ansLines[j] != outLines[j] {
							m[strconv.Itoa(i)] = TestPoint{"WA", "wrong answer"}
							break
						}
					}
				}
			}
			if allOK {
				sumRst(jt.User.Name, jt.Task.Name, "Submitted", "user submitted", m)
			}
			// 清空test目录
			err = os.RemoveAll("test/")
			if err != nil {
				elog.Println(err)
				continue
			}
			err = os.MkdirAll("test/", 0755)
			if err != nil {
				elog.Println(err)
				continue
			}
			log.Println("评测" + jt.Task.Name + ":" + jt.User.Name + "结束")
		case <-stopSignal:
			log.Println("内置评测已停止")
			wg.Done()
			return
		}
	}
}

func sumRst(uname string, tname string, stat string, info string, details map[string]TestPoint) {
	err := rdb.Update(func(tx *buntdb.Tx) error {
		v, e := tx.Get(tname + ":" + uname)
		if e != nil {
			return e
		}
		var ts TaskStat
		e = json.Unmarshal([]byte(v), &ts)
		if e != nil {
			return e
		}
		ts.Stat = stat
		ts.Info = info
		ts.Details = details
		b, e := json.Marshal(ts)
		if e != nil {
			return e
		}
		_, _, e = tx.Set(tname+":"+uname, string(b), nil)
		return e
	})
	if err != nil {
		elog.Println(err)
	}
}
