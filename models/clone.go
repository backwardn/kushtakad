package models

import "time"

type Clone struct {
	ID      int64     `storm:"id,increment,index"`
	FQDN    string    `storm:"index,unique"`
	Depth   int       `storm:"index"`
	Updated time.Time `storm:"index" json:"updated"`
	Created time.Time `storm:"index" json:"created"`
}

func NewClone() *Clone {
	return &Clone{
		Updated: time.Now(),
		Created: time.Now(),
	}
}
