package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

type ServiceAngel struct {
	Auth         *Auth
	AngelCtx     context.Context
	AngelCancel  context.CancelFunc
	SensorCtx    context.Context
	SensorCancel context.CancelFunc
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

func NewServiceAngel(auth *Auth) *ServiceAngel {
	angelCtx, angelCancel := context.WithCancel(context.Background())
	sensorCtx, sensorCancel := context.WithCancel(context.Background())
	angel := &ServiceAngel{
		Auth:         auth,
		AngelCtx:     angelCtx,
		AngelCancel:  angelCancel,
		SensorCtx:    sensorCtx,
		SensorCancel: sensorCancel,
	}
	interuptor(angel.AngelCancel)
	return angel
}

func Run(host, apikey string) {
	auth, err := ValidateAuth(host, apikey)
	if err != nil {
		log.Error("you must pass the cli values -host and -apikey |or| have a valid sensor.json file.")
		log.Fatal(err)
	}
	log.Info(auth)

	svm, err := HTTPServicesConfig(auth.Host, auth.Key)
	if err != nil {
		log.Error("Unable to get the config file for the sensor.")
		log.Fatal(err)
	}
	log.Info(svm)

	angel := NewServiceAngel(auth)
	startSensor(auth, angel.SensorCtx, svm)

	for {
		select {
		case <-angel.AngelCtx.Done(): // if the angel's context is closed
			angel.SensorCtx.Done() // close the sensor's
			log.Info("shutting down angel...done.")
			return
		default:
		}
	}

}
