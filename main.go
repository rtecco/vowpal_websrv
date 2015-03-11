package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/rcrowley/go-metrics"
)

var (
	logfile, modelsDir, pidfile, port string
	nWorkers                          int

	jsonErrorMeter, noModelMeter, predictErrorMeter, readErrorMeter metrics.Meter
	examplesPerRequest                                              metrics.Histogram
	reqTimer                                                        metrics.Timer

	workers *vwWorkers
)

func exitSignalHandler(s os.Signal) {

	log.Println("exiting on", s, "signal")

	os.Remove(pidfile)

	os.Exit(0)
}

func initMetrics() {

	jsonErrorMeter = metrics.NewMeter()
	noModelMeter = metrics.NewMeter()
	predictErrorMeter = metrics.NewMeter()
	readErrorMeter = metrics.NewMeter()

	metrics.Register("json marshalling errors", jsonErrorMeter)
	metrics.Register("no model found", noModelMeter)
	metrics.Register("prediction errors", predictErrorMeter)
	metrics.Register("request read errors", readErrorMeter)

	examplesPerRequest = metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015))

	metrics.Register("examples / request", examplesPerRequest)

	reqTimer = metrics.NewTimer()

	metrics.Register("request timing", reqTimer)
}

func processCmdLine() {

	flag.StringVar(&logfile, "logfile", "", "the logfile location")
	flag.StringVar(&modelsDir, "models", "", "the models directory")
	flag.StringVar(&pidfile, "pidfile", "/tmp/vw_srv.pid", "pidfile path")
	flag.StringVar(&port, "port", "12345", "HTTP port")
	flag.IntVar(&nWorkers, "workers", 4, "the number of worker go-routines to handle requests")

	flag.Parse()

	if isBlank(modelsDir) {

		fmt.Println("--models is required")

		flag.PrintDefaults()

		os.Exit(1)
	}
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {

	reg, ok := metrics.DefaultRegistry.(*metrics.StandardRegistry)

	if !ok {
		log.Fatalln("metricsHandler: metrics.DefaultRegistry type assertion failed")
	}

	json, err := reg.MarshalJSON()

	if err != nil {
		log.Fatalln("metricsHandler: metrics.DefaultRegistry.MarshalJSON:", err)
	}

	httpOkWithJson(w, string(json))
}

func modelsHandler(w http.ResponseWriter, r *http.Request) {

	json, err := json.Marshal(workers.getActiveModels())

	if err != nil {

		log.Println("modelsHandler: json.Marshal:", err)

		jsonErrorMeter.Mark(1)

		http500WithError(w)

		return
	}

	httpOkWithJson(w, string(json))
}

func predictHandler(w http.ResponseWriter, r *http.Request) {

	start := time.Now()
	defer reqTimer.UpdateSince(start)

	query := r.URL.Query()
	modelName := query.Get("m")

	if isBlank(modelName) {

		log.Println("predictHandler: no model specified")

		noModelMeter.Mark(1)

		http400Blank(w)

		return
	}

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {

		log.Println("predictHandler: ioutil.ReadAll:", err)

		readErrorMeter.Mark(1)

		http400Blank(w)

		return
	}

	var exs []string

	err = json.Unmarshal(body, &exs)

	if err != nil {

		log.Println("predictHandler: Unmarshal:", err)

		jsonErrorMeter.Mark(1)

		http500WithError(w)

		return
	}

	examplesPerRequest.Update(int64(len(exs)))

	preds, err := workers.predict(modelName, exs)

	if err != nil {

		log.Println("predictHandler:", err)

		predictErrorMeter.Mark(1)

		http500WithError(w)

		return
	}

	resp, err := json.Marshal(preds)

	if err != nil {

		log.Println("predictHandler: json.Marshal:", err)

		jsonErrorMeter.Mark(1)

		http500WithError(w)

		return
	}

	httpOkWithJson(w, string(resp))
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	processCmdLine()

	setSignalHandler(exitSignalHandler, syscall.SIGTERM, syscall.SIGINT)
	openLogfile(logfile)
	writePidfile(pidfile)

	initMetrics()

	workers = newWorkers(modelsDir)

	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/models", modelsHandler)
	http.HandleFunc("/ping", func(w http.ResponseWriter, req *http.Request) {
		httpOkWithText(w, "pong")
	})

	http.HandleFunc("/p", predictHandler)

	err := http.ListenAndServe(":"+port, nil)

	if err != nil {
		log.Fatalln("ListenAndServe:", err)
	}
}
