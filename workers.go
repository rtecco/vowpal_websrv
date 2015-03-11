package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

const vwExtension = ".vw"

type workerReq struct {
	modelName string
	exs       []string
}

type workerResp struct {
	err   error
	preds []float64
}

type vwModel struct {
	Name         string    `json:"name"`
	LastModified time.Time `json:"last_modified"`
}

type vwWorker struct {
	dir     string
	models  map[string]*VW
	mmtimes map[string]time.Time
	mlock   sync.Mutex
	in      chan *workerReq
	out     chan *workerResp
}

type vwWorkers struct {
	workers []*vwWorker
}

func newWorker(modelsDir string) *vwWorker {

	w := &vwWorker{
		dir:     modelsDir,
		models:  make(map[string]*VW),
		mmtimes: make(map[string]time.Time),
		in:      make(chan *workerReq),
		out:     make(chan *workerResp),
	}

	w.loadAll()

	// reload
	go func() {

		for {
			time.Sleep(30 * time.Second)

			log.Println("vw worker: reloading")

			w.loadAll()
		}
	}()

	// message loop
	go w.run()

	return w
}

func getLastModifiedTime(fullPath string) (time.Time, error) {

	fi, err := os.Stat(fullPath)

	if err != nil {
		return time.Time{}, err
	}

	return fi.ModTime(), nil
}

func (w *vwWorker) getActiveModels() (active []*vwModel) {

	w.mlock.Lock()
	defer w.mlock.Unlock()

	for name, _ := range w.models {
		active = append(active, &vwModel{Name: name, LastModified: w.mmtimes[name]})
	}

	return
}

func (w *vwWorker) shouldLoad(key, fullPath string) bool {

	lastm, err := getLastModifiedTime(fullPath)

	if err != nil {
		log.Fatalln("shouldLoad: getLastModifiedTime:", err)
	}

	lastmStored := w.mmtimes[key]

	return lastm.After(lastmStored)
}

func (w *vwWorker) load(key, fullPath string) {

	vw := NewVW(fullPath)

	w.mlock.Lock()

	oldVW, found := w.models[key]

	// free old
	if found {
		oldVW.Finish()
	}

	w.models[key] = vw

	w.mlock.Unlock()

	//
	// store last modified time

	lastm, err := getLastModifiedTime(fullPath)

	if err != nil {
		log.Fatalln("load: getLastModifiedTime:", err)
	}

	w.mmtimes[key] = lastm
}

func (w *vwWorker) loadAll() {

	//
	// load

	d, err := os.Open(w.dir)
	defer d.Close()

	if err != nil {
		log.Fatalln("loadAll: os.Open:", err)
	}

	fis, err := d.Readdir(-1)

	if err != nil {
		log.Fatalln("loadAll: Readdir:", err)
	}

	//
	// load any new or changed models

	onDisk := make(map[string]struct{})

	for _, fi := range fis {

		if !strings.HasSuffix(fi.Name(), vwExtension) {
			continue
		}

		// the file name, minus the .vw extension, is the key
		key := strings.TrimSuffix(fi.Name(), vwExtension)

		fullPath := path.Join(w.dir, fi.Name())

		if w.shouldLoad(key, fullPath) {

			w.load(key, fullPath)

			log.Println("load: loaded", fi.Name(), "to", key)
		}

		onDisk[key] = struct{}{}
	}

	//
	// delete models that are no longer on disk

	w.mlock.Lock()

	toDelete := []string{}

	for key, _ := range w.models {

		_, found := onDisk[key]

		if !found {
			toDelete = append(toDelete, key)
		}
	}

	for _, key := range toDelete {

		vw := w.models[key]

		vw.Finish()

		delete(w.models, key)

		log.Println("loadAll: deleted", key)
	}

	w.mlock.Unlock()

	log.Println("loadAll: have", len(w.models), "models")
}

func (w *vwWorker) predict(modelName string, exs []string) ([]float64, error) {

	w.in <- &workerReq{modelName: modelName, exs: exs}
	resp := <-w.out

	if resp.err != nil {
		return nil, resp.err
	}

	if len(resp.preds) != len(exs) {
		return nil, errors.New("len(preds) != len(exs)")
	}

	return resp.preds, nil
}

func (w *vwWorker) predictSingle(modelName, ex string) (float64, error) {

	w.in <- &workerReq{modelName: modelName, exs: []string{ex}}
	resp := <-w.out

	if resp.err != nil {
		return 0.0, resp.err
	}

	if len(resp.preds) != 1 {
		return 0.0, errors.New("no prediction")
	}

	return resp.preds[0], nil
}

func (w *vwWorker) run() {

	fmt.Println("running")

	for {
		req := <-w.in

		var resp workerResp

		w.mlock.Lock()

		vw, found := w.models[req.modelName]

		if found {

			for _, ex := range req.exs {
				resp.preds = append(resp.preds, vw.Predict(ex))
			}
		} else {
			resp.err = fmt.Errorf("model %q not found", req.modelName)
		}

		w.mlock.Unlock()

		w.out <- &resp
	}
}

func newWorkers(modelsDir string) *vwWorkers {

	ws := &vwWorkers{}

	for i := 0; i < nWorkers; i++ {
		ws.workers = append(ws.workers, newWorker(modelsDir))
	}

	return ws
}

func (ws *vwWorkers) getActiveModels() []*vwModel {
	return ws.workers[0].getActiveModels()
}

func (ws *vwWorkers) predict(modelName string, exs []string) ([]float64, error) {

	w := ws.workers[rand.Int()%len(ws.workers)]

	return w.predict(modelName, exs)
}
