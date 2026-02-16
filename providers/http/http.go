package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/webitel/wlog"
)

type RecoveryLogger struct{}

type CorsWrapper struct {
	router *mux.Router
}

func (cw *CorsWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO
	if r.Header.Get("Origin") == "" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	}

	if r.Method == "OPTIONS" {
		w.Header().Set(
			"Access-Control-Allow-Methods",
			strings.Join([]string{"GET", "POST", "PUT", "DELETE"}, ", "))

		w.Header().Set(
			"Access-Control-Allow-Headers",
			r.Header.Get("Access-Control-Request-Headers"))
	}

	if r.Method == "OPTIONS" {
		return
	}

	cw.router.ServeHTTP(w, r)
}

func (rl *RecoveryLogger) Println(i ...any) {
	wlog.Error("Please check the std error output for the stack trace")
	wlog.Error(fmt.Sprint(i))
}
