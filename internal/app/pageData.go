package service

import (
	"net/http"
)

type PageData struct {
	Title     string
	Name      string
	Version   string
	IsLogin   bool
	IsAdmin   bool
	IsStarted bool
	IsReg     bool
	IsRk      bool
	StartTime int64
	Duration  int64
	SendFiles map[string]int64
}

func getPageData(w http.ResponseWriter, r *http.Request) PageData {
	var pd PageData
	pd.Title = cfg.Title
	pd.Version = VERSION
	gvm.RLock()
	pd.IsStarted = isStarted
	pd.StartTime = startTime
	pd.Duration = duration
	gvm.RUnlock()
	pd.SendFiles = getSend()

	pd.Name, pd.IsLogin, pd.IsAdmin = checkUser(w, r)
	return pd
}
