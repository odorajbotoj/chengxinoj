package app

type JudgeTask struct {
	User UserData
	Task TaskPoint
}

var judgeQueue = make(chan JudgeTask)

// 评测函数
/*
func judger() {
	for {
		select {
		case jt := <-judgeQueue:
			return
		case <-stopSignal:
			log.Println("内置评测已停止")
			wg.Done()
			return
		}
	}
}
*/
