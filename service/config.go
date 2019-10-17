package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

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

type Auth struct {
	Key  string `json:"key"`
	Host string `json:"host"`
}

type Mapper struct {
	ServiceMap []*ServiceMap
}

type Comms struct {
	Heartbeat time.Time
}

func ParseServices() (*Mapper, error) {

	jsonFile, err := os.Open("services.json")
	if err != nil {
		return nil, err
	}
	fmt.Println("Successfully Opened services.json")
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var mapper *Mapper
	err = json.Unmarshal(byteValue, &mapper)
	if err != nil {
		return nil, err
	}

	return mapper, nil
}

func ValidateAuth(host, apikey string) (*Auth, error) {

	if len(host) > 0 && len(apikey) == 32 {
		return &Auth{Key: apikey, Host: host}, nil
	}

	return ParseAuth()
}

func ParseAuth() (*Auth, error) {
	jsonFile, err := os.Open(auth)
	if err != nil {
		return nil, err
	}
	fmt.Println("Successfully Opened sensor.json")
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var auth *Auth
	err = json.Unmarshal(byteValue, &auth)
	if err != nil {
		return nil, err
	}

	return auth, nil
}

func LastHeartbeat() (time.Time, error) {
	return time.Now(), errors.New("Time unknown")
}

func HTTPServicesConfig(host, key string) ([]*ServiceMap, error) {
	log.Debug("Begin")
	url := host + "/api/v1/config.json"

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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tmpMap []TmpMap
	err = json.Unmarshal(body, &tmpMap)
	if err != nil {
		return nil, err
	}

	log.Info("This is a tmpMap", tmpMap)

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
			log.Infof("Did it decode? %v", tel)
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
			log.Infof("Did it decode? %v", ftp)
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

			err = DownloadDatabase(host, key, httpw.FQDN, newdbname)
			if err != nil {
				log.Fatal(err)
				return nil, err
			}

			db, err := storage.MustDBWithLocationAndName(state.SensorClonesLocation(), newdbname)
			if err != nil {
				log.Fatal(err)
				return nil, err
			}

			httpw.SetHost(host)
			httpw.SetApiKey(key)
			sm.Service = httpw
			sm.DB = db
			svm = append(svm, sm)
			log.Infof("Did the service decode? %v", httpw)
		}
	}

	return svm, nil
}

func DownloadDatabase(host, key, dbname, newdbname string) error {
	log.Debug("Downloading Clone DB %s", dbname)
	url := fmt.Sprintf("%s%s%s", host, "/api/v1/database/", dbname)

	spaceClient := http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))

	resp, err := spaceClient.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fullpath := filepath.Join(state.SensorClonesLocation(), newdbname)
	f, err := os.OpenFile(fullpath, os.O_CREATE|os.O_RDWR, 0660)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(body)
	if err != nil {
		return err
	}

	return nil
}
