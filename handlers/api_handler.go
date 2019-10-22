package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/asdine/storm"
	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
	"go.etcd.io/bbolt"

	"github.com/kushtaka/kushtakad/events"
	"github.com/kushtaka/kushtakad/helpers"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/service"
	"github.com/kushtaka/kushtakad/service/ftp"
	"github.com/kushtaka/kushtakad/service/telnet"
	"github.com/kushtaka/kushtakad/service/webserver"
	"github.com/kushtaka/kushtakad/state"
	"github.com/kushtaka/kushtakad/storage"
)

func GetConfig(w http.ResponseWriter, r *http.Request) {
	var sensor models.Sensor
	var apiKey string
	app, err := state.Restore(r)
	if err != nil {
		app.Render.JSON(w, 404, err)
		return
	}

	token, ok := r.Header["Authorization"]
	if ok && len(token) >= 1 {
		apiKey = token[0]
		apiKey = strings.TrimPrefix(apiKey, "Bearer ")
	}

	err = app.DB.One("ApiKey", apiKey, &sensor)
	if err != nil {
		log.Debug(err)
		app.Render.JSON(w, 200, err)
		return
	}

	svm, err := ServicesConfig(&sensor, app.DB)
	if err != nil {
		log.Debug(err)
		app.Render.JSON(w, 200, err)
		return
	}

	app.Render.JSON(w, http.StatusOK, svm)
	return
}

func GetDatabase(w http.ResponseWriter, r *http.Request) {
	log.Debug("Start")
	app, err := state.Restore(r)
	if err != nil {
		log.Error(err)
		app.Render.JSON(w, 404, err)
		return
	}

	v := mux.Vars(r)
	dbname := v["dbname"]
	db, err := storage.MustDBWithLocationAndName(state.ServerClonesLocation(), dbname)
	if err != nil {
		log.Error(err)
		app.Render.JSON(w, 404, err)
		return
	}
	defer db.Close()

	err = db.Bolt.View(func(tx *bbolt.Tx) error {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, dbname))
		w.Header().Set("Content-Length", strconv.Itoa(int(tx.Size())))
		_, err := tx.WriteTo(w)
		return err
	})

	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Debug("End")

	return
}

func ServicesConfig(s *models.Sensor, db *storm.DB) ([]*service.ServiceMap, error) {
	var svm []*service.ServiceMap
	for _, v := range s.Cfgs {
		switch v.Type {
		case "telnet":
			var tel telnet.TelnetService
			err := mapstructure.Decode(v.Service, &tel)
			if err != nil {
				return nil, err
			}

			sm := &service.ServiceMap{
				Service:    tel,
				SensorName: s.Name,
				Type:       v.Type,
				Port:       strconv.Itoa(v.Port),
			}

			svm = append(svm, sm)
		case "ftp":
			var ftp ftp.FtpService
			err := mapstructure.Decode(v.Service, &ftp)
			if err != nil {
				return nil, err
			}

			sm := &service.ServiceMap{
				Service:    ftp,
				SensorName: s.Name,
				Type:       v.Type,
				Port:       strconv.Itoa(v.Port),
			}

			svm = append(svm, sm)
		case "http":
			var http webserver.HttpService
			err := mapstructure.Decode(v.Service, &http)
			if err != nil {
				return nil, err
			}

			sm := &service.ServiceMap{
				Service: http,
				Type:    v.Type,
				Port:    strconv.Itoa(v.Port),
			}

			svm = append(svm, sm)

		}
	}

	return svm, nil
}

func PostEvent(w http.ResponseWriter, r *http.Request) {
	var sensor models.Sensor
	var apiKey string
	app, err := state.Restore(r)
	if err != nil {
		app.Render.JSON(w, 404, err)
		return
	}

	token, ok := r.Header["Authorization"]
	if ok && len(token) >= 1 {
		apiKey = token[0]
		apiKey = strings.TrimPrefix(apiKey, "Bearer ")
	}

	app.DB.One("ApiKey", apiKey, &sensor)
	if sensor.ApiKey != apiKey {
		app.Render.JSON(w, 404, err)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		app.Render.JSON(w, 404, err)
		return
	}
	defer r.Body.Close()

	var em events.EventManager
	err = json.Unmarshal(b, &em)
	if err != nil {
		log.Error(err)
		app.Render.JSON(w, 404, err)
		return
	}

	err = app.DB.Save(&em)
	if err != nil {
		log.Error(err)
		app.Render.JSON(w, 404, err)
		return
	}

	m := helpers.NewMailer(app.DB, app.Box)
	body := fmt.Sprintf("A [%s] event from the IP [%s] has been detected. Please investigate.", em.State, em.AttackerIP)
	log.Debug(body)
	go func() {
		err := m.SendSensorEvent(em.ID, app.View.URI, "new", body, em.LastNotification)
		if err != nil {
			log.Errorf("SendSensorEvent failed %v", err)
		}
	}()

	app.Render.JSON(w, http.StatusOK, "success")
	return
}
