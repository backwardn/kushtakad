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
	Auth *Auth

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

func NewServiceAngel(auth *Auth, sensor *models.Sensor) *ServiceAngel {
	a := &ServiceAngel{}
	a.AngelCtx, a.AngelCancel = context.WithCancel(context.Background())
	a.SensorCtx, a.SensorCancel = context.WithCancel(context.Background())
	a.Reboot = make(chan bool)
	a.Auth = auth
	a.Sensor = sensor
	interuptor(a.AngelCancel)
	return a
}

func CreateRun(host, apikey string) (*ServiceAngel, error) {
	auth, err := ValidateAuth(host, apikey)
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

func Run(host, apikey string) {
	angel, err := CreateRun(host, apikey)
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

			if sensor.Updated.After(angel.Sensor.Updated) {
				angel.SensorCancel()
			}

		case <-angel.SensorCtx.Done():
			log.Debug("Rebooting...")
			angel, err = CreateRun(host, apikey)
			if err != nil {
				log.Fatal(err)
			}

		case <-angel.AngelCtx.Done(): // if the angel's context is closed
			angel.SensorCancel() // close the sensor's
			log.Info("shutting down angel...done.")
			return
		default:

		}
	}

}
