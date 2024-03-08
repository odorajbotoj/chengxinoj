package app

import "net/http"

func alertAndRedir(w http.ResponseWriter, alert string, redir string) {
	w.Write([]byte(`<!DOCTYPE html><script type="text/javascript">alert("` + alert + `");window.location.replace("` + redir + `");</script>`))
}
