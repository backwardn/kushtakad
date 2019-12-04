package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kushtaka/kushtakad/models"
)

type ServiceAngel struct {
	Auth *models.Auth

	AngelCtx    context.Context
	AngelCancel context.CancelFunc

	SensorCtx    context.Context
	SensorCancel context.CancelFunc

	Sensor *models.Sensor
	Reboot chan bool
}

func interuptor(cancel context.CancelFunc) {
	go func() {
		s := make(chan os.Signal, 1)
		signal.Notify(s, os.Interrupt)
		signal.Notify(s, syscall.SIGTERM)
		select {
		case <-s:
			cancel()
		}
	}()

}

func NewServiceAngel(auth *models.Auth, sensor *models.Sensor) *ServiceAngel {
	a := &ServiceAngel{}
	a.AngelCtx, a.AngelCancel = context.WithCancel(context.Background())
	a.SensorCtx, a.SensorCancel = context.WithCancel(context.Background())
	a.Reboot = make(chan bool)
	a.Auth = auth
	a.Sensor = sensor
	interuptor(a.AngelCancel)
	return a
}

func CreateRun() (*ServiceAngel, error) {
	auth, err := ValidateAuth()
	if err != nil {
		return nil, err
	}

	sensor, err := HTTPSensorHealthCheckAndStatus(auth)
	if err != nil {
		return nil, err
	}

	angel := NewServiceAngel(auth, sensor)

	svm, err := HTTPServicesConfig(auth.Host, auth.Key, angel.SensorCtx)
	if err != nil {
		return nil, err
	}

	startSensor(auth, angel.SensorCtx, svm)

	return angel, nil

}

func Run() {
	angel, err := CreateRun()
	if err != nil {
		log.Fatal(err)
	}

	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			sensor, err := HTTPSensorHealthCheckAndStatus(angel.Auth)
			if err != nil {
				log.Error(err)
			}

			if shouldCancel(angel, sensor) {
				angel.SensorCancel()
			}
		case <-angel.SensorCtx.Done():
			log.Debug("Rebooting...")
			angel, err = CreateRun()
			if err != nil {
				log.Fatal(err)
			}

		case <-angel.AngelCtx.Done(): // if the angel's context is closed
			angel.SensorCancel() // close the sensor's
			log.Info("shutting down angel...done.")
			return
		}
	}

}

func shouldCancel(angel *ServiceAngel, sensor *models.Sensor) bool {

	if sensor == nil {
		log.Error("Unable to cancel sensor as sensor is nil")
		return false
	}

	if !sensor.Updated.After(angel.Sensor.Updated) {
		log.Errorf("On file last update %s : From server last update : %s  ", angel.Sensor.Updated, sensor.Updated)
		return false
	}

	return true
}
