package main

import (
	"flag"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/gobuffalo/packr/v2"
	"github.com/kushtaka/kushtakad/server"
	"github.com/kushtaka/kushtakad/service"
	"github.com/kushtaka/kushtakad/state"
	"github.com/op/go-logging"
)

const empty = ""

var log = logging.MustGetLogger("main")
var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

func Setup(sensor bool) {
	rand.Seed(time.Now().UTC().UnixNano())

	box := packr.New(server.AssetsFolder, "../static")
	err := state.SetupFileStructure(box)
	if err != nil {
		log.Fatalf("Failed to setup file structure : %s", err)
	}

	// setup logfile
	logfile := "server.log"
	if sensor {
		logfile = "sensor.log"
	}

	fp := filepath.Join(state.LogsLocation(), logfile)

	lf, err := os.OpenFile(fp, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	backend1 := logging.NewLogBackend(lf, "", 0)
	backend2 := logging.NewLogBackend(os.Stderr, "", 0)
	backend1Formatter := logging.NewBackendFormatter(backend1, format)
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	logging.SetBackend(backend1Formatter, backend2Formatter)
}

func main() {

	host := flag.String("host", empty, "the hostname of the kushtakad orchestrator server (string)")
	apikey := flag.String("apikey", empty, "the api key of the sensor, create from the kushtaka dashboard. (string)")
	sensor := flag.Bool("sensor", false, "would you like this instance to be a sensor? (bool)")
	flag.Parse()

	Setup(*sensor)

	if *sensor {
		service.Run(*host, *apikey)
	} else {
		server.Run()
	}
}
