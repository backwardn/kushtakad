package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kushtaka/kushtakad/helpers"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/service/ftp"
	"github.com/kushtaka/kushtakad/service/telnet"
	"github.com/kushtaka/kushtakad/service/webserver"
	"github.com/kushtaka/kushtakad/state"
	"github.com/kushtaka/kushtakad/storage"
	"github.com/mitchellh/mapstructure"
)

var data map[string]interface{}

const auth = "sensor.json"
const services = "services.json"
const lastHeartbeat = "lastheartbeat.txt"

type Mapper struct {
	ServiceMap []*ServiceMap
}

type Comms struct {
	Heartbeat time.Time
}

func ParseServices() (*Mapper, error) {

	fp := filepath.Join(helpers.DataDir(), "services.json")
	jsonFile, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	log.Info("Successfully Opened services.json")
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var mapper *Mapper
	err = json.Unmarshal(byteValue, &mapper)
	if err != nil {
		return nil, err
	}

	return mapper, nil
}

func ValidateAuth() (*models.Auth, error) {
	return ParseAuth()
}

func ParseAuth() (*models.Auth, error) {
	fp := filepath.Join(helpers.DataDir(), "sensor.json")
	jsonFile, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	log.Info("Successfully Opened sensor.json")
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var auth *models.Auth
	err = json.Unmarshal(byteValue, &auth)
	if err != nil {
		return nil, err
	}

	return auth, nil
}

func LastHeartbeat() (time.Time, error) {
	return time.Now(), errors.New("Time unknown")
}

func get(key, url string) (*http.Response, error) {
	spaceClient := http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))

	resp, err := spaceClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func HTTPSensorHealthCheckAndStatus(auth *models.Auth) (*models.Sensor, error) {
	url := auth.Host + "/api/v1/sensor.json"

	resp, err := get(auth.Key, url)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var sensor models.Sensor
	err = json.Unmarshal(body, &sensor)
	if err != nil {
		return nil, err
	}

	return &sensor, nil

}

func HTTPServicesConfig(host, key string, ctx context.Context) ([]*ServiceMap, error) {
	url := host + "/api/v1/service_map.json"

	resp, err := get(key, url)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tmpMap []TmpMap
	err = json.Unmarshal(body, &tmpMap)
	if err != nil {
		return nil, err
	}

	var svm []*ServiceMap
	for _, v := range tmpMap {
		switch v.Type {
		case "telnet":
			sm := &ServiceMap{
				Type:       v.Type,
				Port:       v.Port,
				SensorName: v.SensorName,
			}
			var tel telnet.TelnetService
			err := mapstructure.Decode(v.Service, &tel)
			if err != nil {
				return nil, err
			}

			tel.Host = host
			tel.ApiKey = key
			sm.Service = tel
			svm = append(svm, sm)
		case "ftp":
			sm := &ServiceMap{
				Type:       v.Type,
				Port:       v.Port,
				SensorName: v.SensorName,
			}

			var ftp ftp.FtpService
			err := mapstructure.Decode(v.Service, &ftp)
			if err != nil {
				return nil, err
			}

			ftp.Host = host
			ftp.ApiKey = key
			ftp.ConfigureAndRun()
			sm.Service = ftp
			svm = append(svm, sm)
		case "http":
			sm := &ServiceMap{
				Type:       v.Type,
				Port:       v.Port,
				SensorName: v.SensorName,
			}

			var httpw webserver.HttpService
			err := mapstructure.Decode(v.Service, &httpw)
			if err != nil {
				return nil, err
			}

			newdbname := fmt.Sprintf("%d_%s.db", httpw.Port, httpw.FQDN)

			err = HTTPDownloadDatabase(host, key, httpw.FQDN, newdbname)
			if err != nil {
				log.Fatal(err)
				return nil, err
			}

			db, err := storage.MustDBWithLocationAndName(state.SensorClonesLocation(), newdbname)
			if err != nil {
				log.Fatal(err)
				return nil, err
			}

			go func() {
				for {
					select {
					case <-ctx.Done():
						db.Close()
						log.Debugf("Closing cloned DB %s", newdbname)
						return
					}
				}
			}()

			httpw.SetHost(host)
			httpw.SetApiKey(key)
			httpw.Host = host
			httpw.ApiKey = key

			sm.Service = httpw
			sm.DB = db
			svm = append(svm, sm)
		}
	}

	return svm, nil
}

func HTTPDownloadDatabase(host, key, dbname, newdbname string) error {
	log.Debugf("Downloading Clone DB %s and saving as %s", dbname, newdbname)
	url := fmt.Sprintf("%s%s%s", host, "/api/v1/database/", dbname)

	spaceClient := http.Client{
		Timeout: time.Second * 60,
	}

	log.Debugf("The url we are requesting %s", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Error(err)
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))

	resp, err := spaceClient.Do(req)
	if err != nil {
		log.Error(err)
		return err
	}

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("StatusCode want %d got %d", 200, resp.StatusCode)
		log.Error(msg)
		return errors.New(msg)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return err
	}

	fullpath := filepath.Join(state.SensorClonesLocation(), newdbname)
	f, err := os.OpenFile(fullpath, os.O_CREATE|os.O_RDWR, 0660)
	if err != nil {
		log.Error(err)
		return err
	}
	defer f.Close()

	_, err = f.Write(body)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}
