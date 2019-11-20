package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ServiceAngel struct {
	Auth         *Auth
	AngelCtx     context.Context
	AngelCancel  context.CancelFunc
	SensorCtx    context.Context
	SensorCancel context.CancelFunc
	Reboot       chan bool
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
	a := &ServiceAngel{}
	a.Auth = auth
	a.AngelCtx, a.AngelCancel = context.WithCancel(context.Background())
	a.SensorCtx, a.SensorCancel = context.WithCancel(context.Background())
	a.Reboot = make(chan bool)
	interuptor(a.AngelCancel)
	return a
}

func CreateRun(host, apikey string) (*ServiceAngel, error) {
	auth, err := ValidateAuth(host, apikey)
	if err != nil {
		return nil, err
	}

	svm, err := HTTPServicesConfig(auth.Host, auth.Key)
	if err != nil {
		return nil, err
	}

	angel := NewServiceAngel(auth)
	startSensor(auth, angel.SensorCtx, svm)

	return angel, nil

}

func Run(host, apikey string) {
	angel, err := CreateRun(host, apikey)
	if err != nil {
		log.Fatal(err)
	}

	timer := time.NewTimer(time.Second * 3)
	go func() {
		<-timer.C
		angel.SensorCancel()
		fmt.Println("Timer expired")
	}()

	for {
		select {
		case <-angel.SensorCtx.Done():
			log.Debug("Rebooting...")
			angel, err = CreateRun(host, apikey)
			if err != nil {
				log.Fatal(err)
			}

			timer := time.NewTimer(time.Second * 3)
			go func() {
				<-timer.C
				angel.SensorCancel()
				fmt.Println("Timer expired")
			}()

		case <-angel.AngelCtx.Done(): // if the angel's context is closed
			angel.SensorCancel() // close the sensor's
			log.Info("shutting down angel...done.")
			return
		default:
		}
	}

}
