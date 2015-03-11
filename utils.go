package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

func http400Blank(w http.ResponseWriter) {
	http.Error(w, "Bad Request", http.StatusBadRequest)
}

func http500WithError(w http.ResponseWriter) {
	http.Error(w, "Internal Server Error", 500)
}

func httpOkWithJson(w http.ResponseWriter, json string) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	io.WriteString(w, json)
}

func httpOkWithText(w http.ResponseWriter, s string) {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	io.WriteString(w, s)
}

func isBlank(s string) bool {

	if strings.Trim(s, "\r\n ") == "" {
		return true
	}

	return false
}

func openLogfile(logfile string) {

	if logfile != "" {

		lf, err := os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0640)

		if err != nil {
			log.Fatalln("OpenLogfile: os.OpenFile:", err)
		}

		log.SetOutput(lf)
	}
}

func setSignalHandler(handler func(s os.Signal), sig ...os.Signal) {

	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, sig...)

	go func() {
		for s := range signalChan {
			handler(s)
		}
	}()
}

func writePidfile(pidfilePath string) {

	pidString := strconv.Itoa(os.Getpid())

	if err := ioutil.WriteFile(pidfilePath, []byte(pidString), 0644); err != nil {
		log.Fatalln("WritePidfile:", err)
	}
}
