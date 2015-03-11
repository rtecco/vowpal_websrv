## vowpal_websrv

`vowpal_websrv` wraps [Vowpal Wabbit](https://github.com/JohnLangford/vowpal_wabbit) into a webservice and provides some operational niceties for getting your predictive models out into production at scale

### Features

* multiple worker goroutines
* JSON request (via POST) / response
* ability to batch multiple example strings into a single request
* log file / pid file support
* ability to reload models when they change on disk w/o needing to restart the server
* metrics exported via [go-metrics](https://github.com/rcrowley/go-metrics) (request timing, # of errors, histogram of example strings per request, etc.)

### Getting Started

1. You'll need VW installed, either via a `./configure && make && sudo make install` dance or, on OS X, a `brew install vowpal-wabbit`. If you are installing from source, I've found `master` to sometimes be fincky, so try a tagged release.

2. `go get github.com/rtecco/vowpal_websrv`

3. If it doesn't build, check `vw.go` for the various compiler / linker flags. `vowpal_websrv` uses the recently introduced VW C wrapper (`libvw_c_wrapper`) to appease `cgo`.

### Usage

1.) Fire up the server with a simple test model (built with `sample_models/generate.py` against VW 7.10):

```
./vowpal_websrv --workers 1 --models ./sample_models
```

Any file with the `.vw` extension is loaded. The name of the model for the `/p` endpoint below is the base file name (so, if your file is called `v35.vw`, the model name is `v35`). A new model file will be picked up within 30 seconds. Removing a model file will delete the model from memory.

2.) Push some predictions:

```
curl -w "\n" --data-binary @sample_models/request_body.json "http://localhost:12345/p?m=digits"
```

You should see a JSON array of three predictions returned:

```
[0.006233748742058964,0.9963675701101399,0.039889056649459784]
```

Check `--help` to see options for log and pid files and adjusting the number of worker goroutines.

In this case, it classifies the three examples correctly.

### Endpoints

* '/metrics' - GET, return metrics in JSON form provided by [go-metrics](https://github.com/rcrowley/go-metrics)
* '/models' - GET, JSON array of currently loaded models and their last modified timestamps
* '/ping' - GET, returns `pong`, useful for monitoring
* '/p?m=MODEL_NAME' - POST, JSON array of VW examples, returns an array of predictions

### Limitations

* Currently only useful for probability outputs - IOW, the model is assumed to have been trained with logistic loss and a logistic transform will be applied (see `vw.go`).

### TODO

* Ability to get predictions from models trained with other loss functions.
* Use [go-fsnotify/fsnotify](https://github.com/go-fsnotify/fsnotify) to watch the model directory for reloads instead of polling.
* Benchmarking functionality via `wrk`.
* Vagrant builds for Linux.

### License

MIT
