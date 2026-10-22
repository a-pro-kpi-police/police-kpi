package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}
	addr := ":" + envDefault("PORT", "8000")
	logInfof("KPI backend (Go) listening on %s", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if os.Getenv("TZ") == "" {
		_ = os.Setenv("TZ", "UTC")
	}
}
