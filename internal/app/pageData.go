package app

type PageData struct {
	Title     string           // 标题
	Version   string           // 版本
	UserData                   // 用户信息
	IsStarted bool             // 比赛是否开始
	CanReg    bool             // 能否注册用户（非管理员）
	StartTime int64            // 比赛开始时间
	Duration  int64            // 比赛延续时间
	SendFiles map[string]int64 // 下发的文件（文件名-文件大小）
	UserList  []string         // 用户列表
	TaskList  []string         // 任务点列表
	Task      TaskPoint        // 任务点
}

type UserData struct {
	Name    string
	IsLogin bool
	IsAdmin bool
}

func getPageData(ud UserData) PageData {
	var pd PageData
	pd.Title = cfg.Title
	pd.Version = VERSION
	gvm.RLock()
	pd.IsStarted = isStarted
	pd.StartTime = startTime
	pd.Duration = duration
	pd.CanReg = canReg
	gvm.RUnlock()
	pd.UserData = ud
	return pd
}
