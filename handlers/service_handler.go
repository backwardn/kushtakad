package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/asdine/storm"
	"github.com/gorilla/mux"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/service/ftp"
	"github.com/kushtaka/kushtakad/service/telnet"
	"github.com/kushtaka/kushtakad/service/webserver"
	"github.com/kushtaka/kushtakad/state"
)

func DeleteService(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}
	w.Header().Set("Content-Type", "application/json")
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
	}

	var svcCfg models.ServiceCfg
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&svcCfg)
	if err != nil {
		resp = NewResponse("error", "Unable to decode response body", err)
		w.Write(resp.JSON())
		return
	}

	tx, err := app.DB.Begin(true)
	if err != nil {
		resp = NewResponse("error", "Tx can't begin", err)
		w.Write(resp.JSON())
		return
	}
	defer tx.Rollback()

	var sensor models.Sensor
	err = tx.One("ID", svcCfg.SensorID, &sensor)
	if err != nil {
		log.Error(err)
		resp := NewResponse("error", "Sensor id not found, does sensor exist?", err)
		w.Write(resp.JSON())
		return
	}

	for k, v := range sensor.Cfgs {
		if v.UUID == svcCfg.UUID {
			sensor.Cfgs = append(sensor.Cfgs[:k], sensor.Cfgs[k+1:]...)
		}
	}

	// update the time
	sensor.Updated = time.Now()
	err = tx.Update(&sensor)
	if err != nil {
		resp := NewResponse("error", "Unable to update sensor", err)
		w.Write(resp.JSON())
		return
	}

	err = tx.Commit()
	if err != nil {
		resp := NewResponse("error", "Unable to commit tx", err)
		w.Write(resp.JSON())
		return
	}

	msg := fmt.Sprintf("Successfully deleted the [%s] service on port [%d]", svcCfg.Type, svcCfg.Port)
	resp = NewResponse("success", msg, err)
	w.Write(resp.JSON())
	return
}

func PostService(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
	}

	resp := &Response{}
	vars := mux.Vars(r)
	serviceType := vars["type"]
	sensor_id, err := strconv.Atoi(vars["sensor_id"])
	if err != nil {
		resp = NewResponse("error", "Unable to parse sensor id", err)
		w.Write(resp.JSON())
		return
	}

	tx, err := app.DB.Begin(true)
	if err != nil {
		resp = NewResponse("error", "Tx can't begin", err)
		w.Write(resp.JSON())
		return
	}
	defer tx.Rollback()

	var sensor models.Sensor
	tx.One("ID", sensor_id, &sensor)
	if sensor.ID == 0 {
		resp := NewResponse("error", "Sensor id not found, does sensor exist?", err)
		w.Write(resp.JSON())
		return
	}

	cfg, err := CreateService(serviceType, sensor, r, tx)
	if err != nil {
		resp = NewResponse("error", "Unable to create service", err)
		w.Write(resp.JSON())
		return
	}

	sensor.Cfgs = append(sensor.Cfgs, cfg)
	sensor.Updated = time.Now()
	err = tx.Update(&sensor)
	if err != nil {
		resp = NewResponse("error", "Unable to update sensor", err)
		w.Write(resp.JSON())
		return
	}

	err = tx.Commit()
	if err != nil {
		resp := NewResponse("error", "unable to commit tx", err)
		w.Write(resp.JSON())
		return
	}

	resp.Service = &cfg
	resp.Status = "success"
	resp.Message = "Service Saved"
	w.Write(resp.JSON())
}

func CreateService(stype string, sensor models.Sensor, r *http.Request, tx storm.Node) (models.ServiceCfg, error) {
	var err error
	cfg, err := models.NewServiceCfg()
	if err != nil {
		return cfg, fmt.Errorf("Unable to create servicecfg : %w", err)
	}

	switch stype {
	case "telnet":
		var tel telnet.TelnetService
		tel.Prompt = "$ "
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&tel)
		if err != nil {
			return cfg, fmt.Errorf("Unable to decode json : %w", err)
		}

		if tel.Port == 0 {
			return cfg, fmt.Errorf("Port must be specified")
		}

		for _, v := range sensor.Cfgs {
			if v.Port == tel.Port {
				return cfg, fmt.Errorf("Port is already assigned to another service : %w", err)
			}
		}

		cfg.Service = tel
		cfg.SensorID = sensor.ID
		cfg.Type = stype
		cfg.Port = tel.Port
	case "ftp":
		var ftp ftp.FtpService
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&ftp)
		if err != nil {
			return cfg, fmt.Errorf("Unable to decode json : %w", err)
		}

		if ftp.Port == 0 {
			return cfg, fmt.Errorf("Port must be specified")
		}

		for _, v := range sensor.Cfgs {
			if v.Port == ftp.Port {
				return cfg, fmt.Errorf("Port is already assigned to another service : %w", err)
			}
		}

		cfg.Service = ftp
		cfg.SensorID = sensor.ID
		cfg.Type = stype
		cfg.Port = ftp.Port
	case "http":
		var http webserver.HttpService
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&http)
		if err != nil {
			return cfg, fmt.Errorf("Unable to decode json : %w", err)
		}

		if http.Port == 0 {
			return cfg, fmt.Errorf("Port must be specified")
		}

		for _, v := range sensor.Cfgs {
			if v.Port == http.Port {
				return cfg, fmt.Errorf("Port is already assigned to another service : %w", err)
			}
		}

		url, err := url.Parse(http.FQDN)
		if err != nil {
			return cfg, fmt.Errorf("Unable to parse domain name : %w", err)
		}

		http.FQDN = url.Hostname()
		cfg.Service = http
		cfg.SensorID = sensor.ID
		cfg.Type = stype
		cfg.Port = http.Port

	default:
		return cfg, fmt.Errorf("Unable to find service type")
	}
	return cfg, nil
}
