package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/tidwall/buntdb"
)

type JudgeTask struct {
	UserName string
	Task     TaskPoint
}

var judgeQueue = make(chan JudgeTask)

// 评测函数
func judger() {
	// 对windows特判
	var exe string = "outbin.exe"
	if runtime.GOOS == "windows" {
		exe = `.\` + exe
	} else {
		exe = `./` + exe
	}
	for {
		select {
		case jt := <-judgeQueue:
			log.Println("评测" + jt.Task.Name + ":" + jt.UserName)
			// 创建临时目录
			tdn, err := os.MkdirTemp("test", "test-*-temp")
			if err != nil {
				elog.Println(err)
				sumRst(jt.UserName, jt.Task.Name, "Inner Error", err.Error(), nil)
				continue
			}
			// 检查有没有测试点
			ext, err := exists("tasks/" + jt.Task.Name)
			if err != nil {
				elog.Println(err)
				sumRst(jt.UserName, jt.Task.Name, "Inner Error", err.Error(), nil)
				continue
			}
			if !ext {
				elog.Println("评测" + jt.Task.Name + "找不到任务点")
				sumRst(jt.UserName, jt.Task.Name, "Inner Error", "评测"+jt.Task.Name+"找不到任务点", nil)
				continue
			}
			// 获取任务点个数
			fl := getFileList("tasks/" + jt.Task.Name + "/")
			cnt := len(fl)
			if cnt == 0 {
				elog.Println("评测" + jt.Task.Name + "找不到任务点")
				sumRst(jt.UserName, jt.Task.Name, "Inner Error", "评测"+jt.Task.Name+"找不到任务点", nil)
				continue
			}
			if cnt%2 != 0 {
				elog.Println("评测" + jt.Task.Name + "任务点个数不匹配")
				sumRst(jt.UserName, jt.Task.Name, "Inner Error", "评测"+jt.Task.Name+"任务点个数不匹配", nil)
				continue
			}
			cnt /= 2
			// 编译
			// 复制文件
			if jt.Task.SubDir {
				err = copyFile("recvFiles/"+jt.UserName+"/"+jt.Task.Name+"/"+jt.Task.Name+jt.Task.FileType, tdn+"/src"+jt.Task.FileType)
			} else {
				err = copyFile("recvFiles/"+jt.UserName+"/"+jt.Task.Name+jt.Task.FileType, tdn+"/src"+jt.Task.FileType)
			}
			if err != nil {
				elog.Println(err)
				sumRst(jt.UserName, jt.Task.Name, "Inner Error", err.Error(), nil)
				continue
			}
			// 执行编译，生成outbin.exe（为了windows/unix通用）
			var cf []string
			if jt.Task.CFlags != "" { // 对空字符串Split得到[""]，传参进去会导致CE
				cf = strings.Split(jt.Task.CFlags, " ")
			}
			cf = append(cf, "src"+jt.Task.FileType, "-o", "outbin.exe")
			log.Println("编译")
			_, stde, iskilled, _, err := cmdWithTimeout(60000, nil, tdn+"/", jt.Task.CC, cf...)
			if iskilled {
				sumRst(jt.UserName, jt.Task.Name, "CTLE", "compile time limit exceed", nil)
				log.Println("CTLE")
				continue
			}
			if stde != "" {
				sumRst(jt.UserName, jt.Task.Name, "CE", stde, nil)
				log.Println("CE")
				continue
			} else if err != nil {
				sumRst(jt.UserName, jt.Task.Name, "CE", stde, nil)
				log.Println("CE")
				continue
			}
			log.Println("编译完成")
			// 运行
			// 循环，评测每个点
			log.Println("运行")
			var allOK = true
			var m = make(map[int]TestPoint) // 储存每个点的状态
			if jt.Task.FileIO {
				// 文件输入输出
				for i := 1; i <= cnt; i++ {
					// 拷贝输入文件
					err = copyFile(fmt.Sprintf("tasks/%s/%s%d.in", jt.Task.Name, jt.Task.Name, i), tdn+"/"+jt.Task.Name+".in")
					if err != nil {
						elog.Println(err)
						sumRst(jt.UserName, jt.Task.Name, "Inner Error", err.Error(), nil)
						log.Println("Inner Error")
						allOK = false
						break
					}
					// 执行
					log.Println("测试点", i)
					_, runstde, runisk, ti, runerr := cmdWithTimeout(jt.Task.Duration, nil, tdn+"/", exe)
					if runisk { // 超时 TLE
						m[i] = TestPoint{"TLE", "time limit exceed", ti.Milliseconds()}
						log.Println("TLE")
						continue
					}
					if runstde != "" { // 运行出错 RE
						m[i] = TestPoint{"RE", runstde, ti.Milliseconds()}
						log.Println("RE")
						continue
					} else if runerr != nil {
						m[i] = TestPoint{"RE", runerr.Error(), ti.Milliseconds()}
						log.Println("RE")
						continue
					}
					// 比较输出
					ansBytes, err := os.ReadFile(fmt.Sprintf("tasks/%s/%s%d.out", jt.Task.Name, jt.Task.Name, i))
					if err != nil {
						elog.Println(err)
						sumRst(jt.UserName, jt.Task.Name, "Inner Error", err.Error(), m)
						log.Println("Inner Error")
						allOK = false
						break
					}
					outBytes, err := os.ReadFile(fmt.Sprintf(tdn+"/%s.out", jt.Task.Name))
					if err != nil {
						elog.Println(err)
						sumRst(jt.UserName, jt.Task.Name, "Inner Error", err.Error(), m)
						log.Println("Inner Error")
						allOK = false
						break
					}
					// 转字符串，去前后空白，按行分割
					ans := string(ansBytes)
					out := string(outBytes)
					ans = strings.TrimSpace(ans)
					out = strings.TrimSpace(out)
					ansLines := strings.Split(strings.ReplaceAll(ans, "\r\n", "\n"), "\n")
					outLines := strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n")
					if len(ansLines) != len(outLines) {
						if len(ansLines) > len(outLines) {
							m[i] = TestPoint{"WA", "wrong answer (too short)", ti.Microseconds()}
						} else {
							m[i] = TestPoint{"WA", "wrong answer (too long)", ti.Microseconds()}
						}
						log.Println("WA")
						continue
					}
					m[i] = TestPoint{"AC", "accepted", ti.Milliseconds()}
					log.Println("?AC")
					for j := 0; j < len(ansLines); j++ {
						if ansLines[j] != outLines[j] {
							m[i] = TestPoint{"WA", "wrong answer (expect: " + ansLines[j] + ", get: " + outLines[j] + ")", ti.Milliseconds()}
							log.Println("WA!")
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
						elog.Println(err)
						sumRst(jt.UserName, jt.Task.Name, "Inner Error", err.Error(), nil)
						log.Println("Inner Error")
						allOK = false
						break
					}
					// 执行
					log.Println("测试点", i)
					runstdo, runstde, runisk, ti, runerr := cmdWithTimeout(jt.Task.Duration, inpFile, tdn+"/", exe)
					inpFile.Close()
					if runisk { // 超时 TLE
						m[i] = TestPoint{"TLE", "time limit exceed", ti.Microseconds()}
						log.Println("TLE")
						continue
					}
					if runstde != "" { // 运行出错 RE
						m[i] = TestPoint{"RE", runstde, ti.Microseconds()}
						log.Println("RE")
						continue
					} else if runerr != nil {
						m[i] = TestPoint{"RE", runerr.Error(), ti.Microseconds()}
						log.Println("RE")
						continue
					}
					// 比较输出
					ansBytes, err := os.ReadFile(fmt.Sprintf("tasks/%s/%s%d.out", jt.Task.Name, jt.Task.Name, i))
					if err != nil {
						elog.Println(err)
						sumRst(jt.UserName, jt.Task.Name, "Inner Error", err.Error(), m)
						log.Println("Inner Error")
						allOK = false
						break
					}
					// 转字符串，去前后空白，按行分割
					ans := string(ansBytes)
					out := runstdo
					ans = strings.TrimSpace(ans)
					out = strings.TrimSpace(out)
					ansLines := strings.Split(strings.ReplaceAll(ans, "\r\n", "\n"), "\n")
					outLines := strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n")
					if len(ansLines) != len(outLines) {
						if len(ansLines) > len(outLines) {
							m[i] = TestPoint{"WA", "wrong answer (too short)", ti.Microseconds()}
						} else {
							m[i] = TestPoint{"WA", "wrong answer (too long)", ti.Microseconds()}
						}
						log.Println("WA")
						continue
					}
					m[i] = TestPoint{"AC", "accepted", ti.Microseconds()}
					log.Println("?AC")
					for j := 0; j < len(ansLines); j++ {
						if ansLines[j] != outLines[j] {
							m[i] = TestPoint{"WA", "wrong answer (expect: " + ansLines[j] + ", get: " + outLines[j] + ")", ti.Microseconds()}
							log.Println("WA!")
							break
						}
					}
				}
			}
			log.Println("运行完成")
			if allOK {
				s := "AC"
				for _, v := range m {
					if v.Stat != "AC" {
						s = v.Stat
						break
					}
				}
				sumRst(jt.UserName, jt.Task.Name, s, "user submitted", m)
			}
			// 清空temp目录
			err = os.RemoveAll(tdn)
			if err != nil {
				elog.Println(err)
				continue
			}
			log.Println("评测" + jt.Task.Name + ":" + jt.UserName + "结束")
		case <-stopSignal:
			log.Println("内置评测已停止")
			wg.Done()
			return
		}
	}
}

// 限时执行命令
func cmdWithTimeout(tout int, inp io.Reader, dir string, cmd string, args ...string) (string, string, bool, time.Duration, error) {
	var stdo, stde bytes.Buffer
	var isKilled bool
	var err error
	c := exec.Command(cmd, args...)
	c.Stdin = inp
	c.Stdout = &stdo
	c.Stderr = &stde
	c.Dir = dir
	var t time.Duration
	done := make(chan error)
	err = c.Start()
	if err != nil {
		return "", "", false, 0, err
	}
	sT := time.Now()
	after := time.After(time.Duration(tout) * time.Millisecond)
	go func() {
		done <- c.Wait()
	}()
	select {
	case <-after:
		c.Process.Signal(syscall.SIGINT)
		t = time.Since(sT)
		time.Sleep(100 * time.Millisecond)
		c.Process.Kill()
		isKilled = true
	case e := <-done:
		t = time.Since(sT)
		isKilled = false
		err = e
	case <-stopSignal:
		c.Process.Signal(syscall.SIGINT)
		time.Sleep(100 * time.Millisecond)
		c.Process.Kill()
		isKilled = true
	}
	stdout := stdo.String()
	stderr := stde.String()
	return stdout, stderr, isKilled, t, err
}

// 生成TaskStat结果并存储至RDB
func sumRst(uname string, tname string, stat string, info string, details map[int]TestPoint) {
	err := rdb.Update(func(tx *buntdb.Tx) error {
		var ts TaskStat
		v, e := tx.Get(tname + ":" + uname)
		if e != nil {
			if e != buntdb.ErrNotFound {
				return e
			} else {
				ts.Judge = true
			}
		} else {
			e = json.Unmarshal([]byte(v), &ts)
			if e != nil {
				return e
			}
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

// 重评测
func reJudgeTask(task TaskPoint) {
	ul := getUserList()
	for _, i := range ul {
		var ise bool
		if task.SubDir {
			ise, _ = exists("recvFiles/" + i + "/" + task.Name + "/" + task.Name + task.FileType)
		} else {
			ise, _ = exists("recvFiles/" + i + "/" + task.Name + task.FileType)
		}
		if ise {
			judgeQueue <- JudgeTask{i, task}
		}
	}
}
