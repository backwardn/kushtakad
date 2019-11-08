package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("event_manager")

const (
	newEvent     = "new"
	ongoingEvent = "ongoing"
	trusted      = "trusted"
)

type EventManager struct {
	ID              int64  `storm:"id,increment,index"`
	State           string `json:"state"` // new, ongoing, trusted
	Type            string `storm:"index" json:"type"`
	AttackerNetwork string `storm:"index"`
	AttackerIP      string `storm:"index"`
	EventType       interface{}
	Created         time.Time   `storm:"index"`
	mu              *sync.Mutex `json:"-"`
}

type EventType interface {
	Is() string
}

type EventSensor struct {
	SensorID     int64  `json:"sensor_id"`
	Type         string `json:"type"`
	Port         int    `json:"port"`
	AttackerPort string `storm:"index" json:"attacker_port"`
}

func TypeID(em EventManager) int64 {
	switch em.Type {
	case "sensor":
		var e EventSensor
		j, _ := json.Marshal(em.EventType)
		json.Unmarshal([]byte(j), &e)
		return e.SensorID
	case "token":
		var e EventToken
		j, _ := json.Marshal(em.EventType)
		json.Unmarshal([]byte(j), &e)
		return e.TokenID
	}
	return 0
}

func MapToEventSensor(em EventManager) EventSensor {
	var e EventSensor
	switch em.Type {
	case "sensor":
		j, _ := json.Marshal(em.EventType)
		json.Unmarshal([]byte(j), &e)
	}
	return e
}

func (em EventManager) SetSensorID(i int64) {
	switch e := em.EventType.(type) {
	case *EventSensor:
		e.SensorID = i
		em.EventType = e
	case *EventToken:
		e.TokenID = i
		em.EventType = e
	}
}

func (e *EventSensor) Is() string {
	return e.Type
}

type EventToken struct {
	TokenID int64  `json:"token_id"`
	Type    string `json:"type"`
}

func (e *EventToken) MyID() int64 {
	return e.TokenID
}

func (e *EventToken) Is() string {
	return e.Type
}

func (em *EventManager) AddMutex() {
	em.mu = &sync.Mutex{}
}

func NewSensorEventManager(network, ip string, e *EventSensor) *EventManager {
	return NewEventManager("sensor", network, ip, e)
}

func NewTokenEventManager(network, ip string, e *EventToken) *EventManager {
	return NewEventManager("token", network, ip, e)
}

func NewEventManager(typ, network, ip string, et EventType) *EventManager {
	return &EventManager{
		Type:            typ,
		EventType:       et,
		AttackerNetwork: network,
		AttackerIP:      ip,
		Created:         time.Now(),
		mu:              &sync.Mutex{},
	}
}

func (em *EventManager) SendEvent(state, host, key string) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.State = state
	url := host + "/api/v1/event.json"

	spaceClient := http.Client{
		Timeout: time.Second * 5,
	}

	json, err := json.Marshal(em)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(json))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))

	resp, err := spaceClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Debug(body)

	return nil

}

func (em *EventManager) SetState(db *storm.DB) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.State = "new"

	var targets []EventManager
	past := time.Now().Add(time.Second * time.Duration(-30))
	err := db.Select(
		q.Eq("AttackerIP", em.AttackerIP),
		q.Gte("Created", past)).Find(&targets)

	if err != nil {
		log.Errorf("Targets are %v", err)
	}

	if len(targets) > 0 {
		em.State = "ongoing"
	}

}
