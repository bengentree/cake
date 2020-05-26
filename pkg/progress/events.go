package progress

import (
	"encoding/json"
	"fmt"
	"time"

	natsd "github.com/nats-io/nats-server/server"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

// StatusEvent is type that is used for pub/sub of events
type StatusEvent struct {
	Type  string `json:"type"`
	Msg   string `json:"msg"`
	Level string `json:"level"`
}

// String of StatusEvent
func (s StatusEvent) String() string {
	return fmt.Sprintf(`{"type": "%v", "msg": "%v", "level": "%v"}`, s.Type, s.Msg, s.Level)
}

// ToLogrusFields is a helper for the logrus library
func (s StatusEvent) ToLogrusFields() logrus.Fields {
	var t StatusEvent
	err := json.Unmarshal([]byte(s.Msg), &t)
	if err == nil {
		return logrus.Fields{
			"type":  t.Type,
			"msg":   t.Msg,
			"level": t.Level,
		}
	}
	return logrus.Fields{"type": s.Type, "msg": s.Msg, "level": s.Level}
}

// Events interface for publish/subscribing to events
type Events interface {
	Publish(*StatusEvent) error
	Subscribe(func(*StatusEvent)) error
}
type natsPubSub struct {
	subj string
	conn *nats.EncodedConn
}

// Publish an event to a subject
func (n *natsPubSub) Publish(p *StatusEvent) error {
	return n.conn.Publish(n.subj, p)
}

// Subscribe to a subject
func (n *natsPubSub) Subscribe(fn func(*StatusEvent)) error {
	_, err := n.conn.Subscribe(n.subj, fn)
	return err
}

// NewNatsPubSub returns an Events interface for Publishing and Subscribing to Events
func NewNatsPubSub(url string, subj string) (Events, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	c, err := nats.NewEncodedConn(nc, "json")
	if err != nil {
		return nil, err
	}
	return &natsPubSub{subj: subj, conn: c}, nil
}

// RunServer starts an embedded nats server
func RunServer() error {
	ns := natsd.New(&natsd.Options{})
	go ns.Start()
	if !ns.ReadyForConnections(10 * time.Second) {
		return fmt.Errorf("not able to start")
	}
	return nil
}
