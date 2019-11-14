package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/asdine/storm"
	"github.com/gobuffalo/packr/v2"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/server"
	"github.com/kushtaka/kushtakad/service"
	"github.com/kushtaka/kushtakad/state"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
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

func tryResetAdmin(user, pass string) (bool, error) {
	if len(user) == 0 && len(pass) == 0 {
		return false, nil
	}

	if len(user) > 0 && len(pass) == 0 {
		return true, errors.Errorf("Email was set but Password is missing, both are required.")
	}

	if len(pass) > 0 && len(user) == 0 {
		return true, errors.Errorf("Password was set but Email is missing, both are required.")
	}

	if len(pass) < 12 {
		return true, errors.Errorf("Password must be at least 12 characters.")
	}

	if len(user) < 4 {
		return true, errors.Errorf("Email must be at least 12 characters.")
	}

	db, err := storm.Open(state.DbLocation())
	if err != nil {
		return true, errors.Errorf("Failed to open database : %s", err)
	}
	defer db.Close()

	u := &models.User{
		ID:              1,
		Email:           user,
		Password:        pass,
		PasswordConfirm: pass,
	}

	err = u.ValidateCreateUser()
	if err != nil {
		return true, errors.Errorf("Unable to validate user > %v", err)
	}

	u.HashPassword()

	tx, err := db.Begin(true)
	if err != nil {
		return true, errors.Errorf("Unable to begin tx > %v", err)
	}

	err = tx.Save(user)
	if err != nil {
		err := tx.Update(u)
		if err != nil {
			return true, errors.Errorf("Unable to save or update tx > %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return true, errors.Errorf("Unable to commit tx > %v", err)
	}
	return true, nil
}

func main() {

	adminuser := flag.String("email", empty, "set the email of the kushtakad admin user (string)")
	adminpass := flag.String("password", empty, "set the password of the kushtakad admin user (string)")
	host := flag.String("host", empty, "the hostname of the kushtakad orchestrator server (string)")
	apikey := flag.String("apikey", empty, "the api key of the sensor, create from the kushtaka dashboard. (string)")
	sensor := flag.Bool("sensor", false, "would you like this instance to be a sensor? (bool)")
	flag.Parse()

	Setup(*sensor)
	didTry, err := tryResetAdmin(*adminuser, *adminpass)
	if err != nil {
		log.Fatalf("Failed to reset/setup admin email and password > %v", err)
	} else if didTry && err == nil {
		fmt.Println("Admin email and password reset/setup was succesful.")
		fmt.Println("Please start kushtakad normally.")
		return
	}

	if *sensor {
		service.Run(*host, *apikey)
	} else {
		server.Run()
	}
}
