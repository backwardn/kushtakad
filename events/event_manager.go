package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("event_manager")

const newEvent = "new"
const ongoingEvent = "ongoing"

type EventManager struct {
	ID                 int64  `storm:"id,increment,index"`
	State              string `json:"type"` // new, ongoing
	AttackerNetwork    string
	AttackerIP         string
	SensorID           int64 `json:"sensor_id"`
	SensorType         string
	SensorPort         int
	EventStart         time.Time   `storm:"index"`
	AttackerLastProbed time.Time   `storm:"index"`
	LastNotification   time.Time   `json:"-"`
	mu                 *sync.Mutex `json:"-"`
}

func NewEventManager(st string, sp int, sid int64) *EventManager {
	t := time.Now()
	return &EventManager{
		mu:         &sync.Mutex{},
		SensorID:   sid,
		SensorPort: sp,
		SensorType: st,
		EventStart: t,
	}
}

func (em *EventManager) SendEvent(state, host, key string, addr net.Addr) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	log.Debugf("host %s key %s", host, key)
	t := time.Now()
	em.State = state
	em.AttackerNetwork = addr.Network()
	em.AttackerIP = addr.String()
	em.AttackerLastProbed = time.Now()
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

	em.LastNotification = t

	return nil

}
