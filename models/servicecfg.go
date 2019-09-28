package models

import (
	"context"
	"errors"
	"net"

	"github.com/google/uuid"
)

type ServiceCfg struct {
	UUID     uuid.UUID `storm:"index,unique" json:"uuid"`
	SensorID int64     `storm:"index" json:"sensor_id"`
	Port     int       `storm:"index" json:"port"`
	Type     string    `storm:"index" json:"type"`

	Service interface{}
}

func NewServiceCfg() (ServiceCfg, error) {
	var cfg ServiceCfg
	uuid := uuid.New()
	if len(uuid) != 16 {
		return cfg, errors.New("UUID did not create, must fail")
	}

	cfg.UUID = uuid

	return cfg, nil

}

type Service interface {
	Handle(ctx context.Context, conn net.Conn) error
}
