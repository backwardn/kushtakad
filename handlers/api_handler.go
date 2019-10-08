package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/asdine/storm"
	"github.com/mitchellh/mapstructure"

	"github.com/kushtaka/kushtakad/events"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/service"
	"github.com/kushtaka/kushtakad/service/ftp"
	"github.com/kushtaka/kushtakad/service/telnet"
	"github.com/kushtaka/kushtakad/state"
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

	app.DB.One("ApiKey", apiKey, &sensor)
	// TODO: add constant time compare
	// update: not needed, handled in middleware
	if sensor.ApiKey != apiKey {
		log.Debug("Api key does NOT match")
		app.Render.JSON(w, 404, err)
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

	// TODO: add constant time compare
	// update: not needed, handled in middleware
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

	app.Render.JSON(w, http.StatusOK, "success")
	return
}
