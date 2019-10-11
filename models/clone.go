package models

import "time"

type Clone struct {
	ID      int64     `storm:"id,increment,index"`
	FQDN    string    `storm:"fqdn"`
	Name    string    `storm:"index,unique" json:"name"`
	Note    string    `storm:"index" json:"note"`
	Updated time.Time `storm:"index" json:"updated"`
	Created time.Time `storm:"index" json:"created"`
}
