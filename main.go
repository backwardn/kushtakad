package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/asdine/storm"
	"github.com/gobuffalo/packr/v2"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/server/server"
	"github.com/kushtaka/kushtakad/service/service"
	"github.com/kushtaka/kushtakad/state"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
)

const empty = ""

var log = logging.MustGetLogger("main")
var format = logging.MustStringFormatter(
	`%{color}%{time:2006-01-02 15:04:05.000000000 MST -07:00} %{id:03x} %{level:.4s} ▶ %{shortfunc} %{color:reset} %{message}`,
)

func Setup(sensor bool) {
	rand.Seed(time.Now().UTC().UnixNano())
	// setup logfile
	logfile := "server.log"
	if sensor {
		logfile = "sensor.log"
	}

	box := packr.New(state.AssetsFolder, "../static")
	err := state.SetupFileStructure(box)
	if err != nil {
		log.Fatalf("Failed to setup file structure : %s", err)
	}

	fp := filepath.Join(state.LogsLocation(), logfile)

	lf, err := os.OpenFile(fp, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
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
		return true, errors.Errorf("Email must be at least 4 characters.")
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
	defer tx.Rollback()

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

func createSensorCfg(apikey, host string) error {
	sensorCfgPath := state.DataDirLocation()

	fmt.Printf("The path for the sensor.json file is %v\n", sensorCfgPath)

	if _, err := os.Stat(sensorCfgPath); os.IsNotExist(err) {
		err = os.MkdirAll(sensorCfgPath, 0744)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to make directory %s", sensorCfgPath))
		}
	}
	a := &models.Auth{
		Host: host,
		Key:  apikey,
	}

	b, err := json.MarshalIndent(a, "", " ")
	if err != nil {
		return err
	}

	fp := path.Join(sensorCfgPath, "sensor.json")
	err = ioutil.WriteFile(fp, b, 0744)
	if err != nil {
		return err
	}

	return nil
}

func trySensorCfg(apikey, host string) (bool, error) {

	if len(apikey) == 0 && len(host) == 0 {
		return false, nil
	}

	if len(apikey) > 0 && len(host) == 0 {
		return true, errors.Errorf("ApiKey was set but Host is missing, both are required.")
	}

	if len(host) > 0 && len(apikey) == 0 {
		return true, errors.Errorf("Host was set but ApiKey is missing, both are required.")
	}

	if len(apikey) != 64 {
		return true, errors.Errorf("ApiKey must be at 32 characters.")
	}

	if len(host) < 4 {
		return true, errors.Errorf("Host must be at least 4 characters.")
	}

	createSensorCfg(apikey, host)

	return true, nil
}

func main() {

	// server mode flags
	email := flag.String("email", empty, "set the email of the kushtakad admin user (string)")
	password := flag.String("password", empty, "set the password of the kushtakad admin user (string)")
	serv := flag.Bool("server", false, "would you like this instance to be a server? (bool)")

	// sensor mode flags
	host := flag.String("host", empty, "the hostname of the kushtakad orchestrator server (string)")
	apikey := flag.String("apikey", empty, "the api key of the sensor, create from the kushtaka dashboard. (string)")
	sensor := flag.Bool("sensor", false, "would you like this instance to be a sensor? (bool)")
	flag.Parse()

	Setup(*sensor)

	tryReset, err := tryResetAdmin(*email, *password)
	if err != nil {
		log.Fatalf("Failed to set the admin email and password > %v", err)
	} else if tryReset && err == nil {
		fmt.Println("Successfuly set the admin email and password.")
		return
	}

	trySensorCfg, err := trySensorCfg(*apikey, *host)
	if err != nil {
		log.Fatalf("Failed to setup sensor.json file > %v", err)
	} else if trySensorCfg && err == nil {
		fmt.Println("Successfully setup sensor.json file.")
		return
	}

	if *sensor {
		service.Run()
	} else if *serv {
		server.Run()
	} else if os.Getenv("KUSHTAKA_ENV") == "development" {
		server.Run()
	} else {
		fmt.Println("You can pass the (-apikey && -host) flags to configure the kushtakd sensor.json file.")
		fmt.Println("Or you can pass the (-email && -password) flags to reset/setup kushtakd's server admin user.")
		fmt.Println("Then you must specify the correct flags (-server | -sensor) in order to start kushtakad in the desired mode.")
		return
	}
}
