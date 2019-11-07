package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("event_manager")

const newEvent = "new"
const ongoingEvent = "ongoing"

type EventManager struct {
	ID              int64  `storm:"id,increment,index"`
	State           string `json:"state"` // new, ongoing
	Type            string `json:"type"`
	AttackerNetwork string `storm:"index"`
	AttackerIP      string `storm:"index"`
	AttackerPort    string `storm:"index"`
	SensorID        int64  `json:"sensor_id"`
	SensorType      string
	SensorPort      int
	Created         time.Time   `storm:"index"`
	mu              *sync.Mutex `json:"-"`
}

func (em *EventManager) AddMutex() {
	em.mu = &sync.Mutex{}
}

func NewSensorEventManager(st string, sp int, sid int64) *EventManager {
	t := time.Now()
	return &EventManager{
		mu:         &sync.Mutex{},
		Type:       "sensor",
		SensorID:   sid,
		SensorPort: sp,
		SensorType: st,
		Created:    t,
	}
}

func NewEventManager(st string, sp int, sid int64) *EventManager {
	t := time.Now()
	return &EventManager{
		mu:         &sync.Mutex{},
		SensorID:   sid,
		SensorPort: sp,
		SensorType: st,
		Created:    t,
	}
}

func (em *EventManager) SendEvent(state, host, key string, addr net.Addr) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	log.Debugf("host %s key %s", host, key)
	s := strings.Split(addr.String(), ":")
	em.State = state
	em.AttackerNetwork = addr.Network()
	em.AttackerIP = s[0]
	em.AttackerPort = s[1]
	em.Created = time.Now()
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
