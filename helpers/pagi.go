package helpers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Pagi struct {
	Offset      int
	OffsetSize  int
	Limit       int
	Page        int
	PageTotal   int
	GrandTotal  int
	PrevPage    int
	PrevEnabled bool
	PrevLink    string
	NextPage    int
	NextEnabled bool
	NextLink    string
	BaseURI     string
}

func NewPagi() *Pagi {
	p := &Pagi{}
	p.Page = 1
	p.Offset = 0
	p.Limit = 5
	p.PrevPage = 1
	p.PrevEnabled = false
	p.NextPage = 1
	p.NextEnabled = false
	return p
}

func (this *Pagi) SetupButtons() {
	this.NextEnabled = false
	this.PrevEnabled = false

	if this.Page > 1 {
		this.PrevEnabled = true
		this.PrevPage = this.Page - 1
		this.PrevLink = fmt.Sprintf("%v/page/%v/limit/%v", this.BaseURI, this.PrevPage, this.Limit)
	}

	calcdTotal := int(this.Limit * this.Page)

	if this.GrandTotal > calcdTotal {
		this.NextEnabled = true
		this.NextPage = this.Page + 2
		this.NextLink = fmt.Sprintf("%v/page/%v/limit/%v", this.BaseURI, this.NextPage, this.Limit)
	}
}

func (this *Pagi) CalcOffset() {

	if this.Page >= 1 {
		this.Offset = (this.Page - 1) * this.Limit
	}

	log.Debugf("Offset:%v Page:%v Limit:%v", this.Offset, this.Page, this.Limit)
}

func (this *Pagi) LimitIsValid() bool {
	switch this.Limit {
	case 5:
		return true
	case 10:
		return true
	case 25:
		return true
	case 50:
		return true
	case 100:
		return true
	case 1000:
		return true
	}
	return false
}

func (this *Pagi) PageIsValid() bool {
	if this.Page >= 1 {
		return true
	}
	return false
}

func (this *Pagi) Configure(grandtotal int, r *http.Request) error {

	vars := mux.Vars(r)
	page, _ := strconv.Atoi(vars["pid"])
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(vars["oid"])
	if limit < 1 {
		limit = 10
	}

	this.Page = page
	this.Limit = limit
	this.GrandTotal = grandtotal

	if this.LimitIsValid() == false {
		err := errors.New("Limit isn't valid")
		return err
	}

	if this.PageIsValid() == false {
		err := errors.New("Page isn't valid")
		return err
	}

	if len(this.BaseURI) < 2 {
		err := errors.New("BasePath not set")
		return err
	}

	this.CalcOffset()
	this.SetupButtons()

	return nil
}

func (this *Pagi) ConfigureMan(grandtotal, page, limit int, r *http.Request) error {

	this.Page = page
	this.Limit = limit
	this.GrandTotal = grandtotal

	if this.LimitIsValid() == false {
		err := errors.New("Limit isn't valid")
		return err
	}

	if this.PageIsValid() == false {
		err := errors.New("Page isn't valid")
		return err
	}

	if len(this.BaseURI) < 2 {
		err := errors.New("BasePath not set")
		return err
	}

	this.CalcOffset()
	this.SetupButtons()

	return nil
}
