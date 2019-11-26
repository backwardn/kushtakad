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
	"github.com/kushtaka/kushtakad/state"
	"github.com/kushtaka/kushtakad/server/server"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
)

const empty = ""

var log = logging.MustGetLogger("main")
var format = logging.MustStringFormatter(
	`%{color}%{time:2006-01-02 15:04:05.000000000 MST -07:00} %{id:03x} %{level:.4s} ▶ %{shortfunc} %{color:reset} %{message}`,
)

func Setup() {
	rand.Seed(time.Now().UTC().UnixNano())
	// setup logfile
	logfile := "server.log"

	box := packr.New(state.AssetsFolder, "../static")
	err := state.SetupFileStructure(box)
	if err != nil {
		log.Fatalf("Failed to setup file structure : %s", err)
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

func main() {

	adminuser := flag.String("email", empty, "set the email of the kushtakad admin user (string)")
	adminpass := flag.String("password", empty, "set the password of the kushtakad admin user (string)")
	flag.Parse()

	Setup()
	didTry, err := tryResetAdmin(*adminuser, *adminpass)
	if err != nil {
		log.Fatalf("Failed to reset/setup admin email and password > %v", err)
	} else if didTry && err == nil {
		fmt.Println("Admin email and password reset/setup was succesful.")
		fmt.Println("Please start kushtakad normally.")
		return
	}

	server.Run()
}
