package app

import (
	"net/http"
)

type PageData struct {
	Title   string
	Version string
	UserData
	IsStarted bool
	IsReg     bool
	IsRk      bool
	CanReg    bool
	CanSubmit bool
	StartTime int64
	Duration  int64
	SendFiles map[string]int64
	Users     []string
}

type UserData struct {
	Name    string
	IsLogin bool
	IsAdmin bool
}

func getPageData(r *http.Request, ud UserData) PageData {
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

	if pd.IsLogin {
		pd.SendFiles = getFileList("send/")
	}
	if pd.IsAdmin {
		pd.Users = getUserList()
	}
	return pd
}
