package app

import "net/http"

func fRk(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		//
	} else {
		// 400
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}
}
