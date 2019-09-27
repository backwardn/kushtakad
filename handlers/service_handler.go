package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/asdine/storm"
	"github.com/gorilla/mux"
	"github.com/kushtaka/kushtakad/models"
	"github.com/kushtaka/kushtakad/service/telnet"
	"github.com/kushtaka/kushtakad/state"
)

func DeleteService(w http.ResponseWriter, r *http.Request) {
	resp := &Response{}
	w.Header().Set("Content-Type", "application/json")
	app, err := state.Restore(r)
	if err != nil {
		log.Fatal(err)
	}

	var scfgFinder models.ServiceCfg
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&scfgFinder)
	if err != nil {
		resp = NewResponse("error", "Unable to decode response body", err)
		w.Write(resp.JSON())
		return
	}
	log.Debug(scfgFinder.ServiceID)

	tx, err := app.DB.Begin(true)
	if err != nil {
		resp = NewResponse("error", "Tx can't begin", err)
		w.Write(resp.JSON())
		return
	}
	defer tx.Rollback()

	var scfg models.ServiceCfg
	err = tx.One("ServiceID", scfgFinder.ServiceID, &scfg)
	if err != nil {
		resp := NewResponse("error", "Scfg does not exist?", err)
		w.Write(resp.JSON())
		return
	}

	var sensor models.Sensor
	err = tx.One("ID", scfg.SensorID, &sensor)
	if err != nil {
		log.Error(err)
		resp := NewResponse("error", "Sensor id not found, does sensor exist?", err)
		w.Write(resp.JSON())
		return
	}

	for k, v := range sensor.Cfgs {
		if v.ServiceID == scfg.ServiceID {
			sensor.Cfgs = append(sensor.Cfgs[:k], sensor.Cfgs[k+1:]...)
		}
	}

	err = tx.Update(&sensor)
	if err != nil {
		resp := NewResponse("error", "Unable to update sensor", err)
		w.Write(resp.JSON())
		return
	}

	switch scfg.Type {
	case "telnet":
		var tel telnet.TelnetService
		err := tx.One("ID", scfg.ServiceID, &tel)
		if err != nil {
			resp := NewResponse("error", "Unable to find telnet service", err)
			w.Write(resp.JSON())
			return
		}

		err = tx.DeleteStruct(&tel)
		if err != nil {
			resp := NewResponse("error", "Unable to delete telnet struct", err)
			w.Write(resp.JSON())
			return
		}

		err = tx.DeleteStruct(&scfg)
		if err != nil {
			resp := NewResponse("error", "Unable to delete ServiceCfg struct", err)
			w.Write(resp.JSON())
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		resp := NewResponse("error", "Unable to commit tx", err)
		w.Write(resp.JSON())
		return
	}

	msg := fmt.Sprintf("Successfully delete the [%s] service on port [%d]", scfg.Type, scfg.Port)
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
	sensorId, err := strconv.Atoi(vars["sensor_id"])
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
	tx.One("ID", sensorId, &sensor)
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

	err = tx.Save(&cfg)
	if err != nil {
		resp = NewResponse("error", "Unable to save service configuration", err)
		w.Write(resp.JSON())
		return
	}

	sensor.Cfgs = append(sensor.Cfgs, cfg)
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
	var cfg models.ServiceCfg
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

		err = tx.Save(&tel)
		if err != nil {
			return cfg, fmt.Errorf("Unable to save telnet service : %w", err)
		}

		cfg.ServiceID = tel.ID
		cfg.SensorID = sensor.ID
		cfg.Type = stype
		cfg.Port = tel.Port

	default:
		return cfg, fmt.Errorf("Unable to find service type")
	}
	return cfg, nil
}
