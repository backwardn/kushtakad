package models

import (
	"context"
	"net"
)

type ServiceCfg struct {
	ID        int64  `storm:"id,increment,index" json:"ID"`
	SensorID  int64  `storm:"index" json:"sensorId"`
	ServiceID int64  `storm:"index" json:"serviceId"`
	Port      int    `storm:"index" json:"port"`
	Type      string `storm:"index" json:"type"`

	Service *Service
}

type Service interface {
	Handle(ctx context.Context, conn net.Conn) error
}
