package models

import (
	"context"
	"net"
)

type ServiceCfg struct {
	SensorID int64  `storm:"index" json:"sensorId"`
	Port     int    `storm:"index" json:"port"`
	Type     string `storm:"index" json:"type"`

	Service interface{}
}

type Service interface {
	Handle(ctx context.Context, conn net.Conn) error
}
