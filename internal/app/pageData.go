package app

type PageData struct {
	Title   string
	Version string
	UserData
	IsStarted bool
	CanReg    bool
	CanSubmit bool
	StartTime int64
	Duration  int64
	SendFiles map[string]int64
	UserList  []string
	TaskList  []string
	Task      TaskPoint
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
	pd.CanSubmit = canSubmit
	gvm.RUnlock()
	pd.UserData = ud
	return pd
}
