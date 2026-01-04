package nats

import (
	"errors"
	"time"

	"github.com/nats-io/nats.go"
)

func NewConnection(dsn string) (*nats.Conn, error) {
	nc, err := nats.Connect(dsn)
	if err != nil {
		return nil, err
	}

	if err := nc.FlushTimeout(1 * time.Second); err != nil {
		return nil, errors.New("not connected")
	}

	return nc, nil
}
