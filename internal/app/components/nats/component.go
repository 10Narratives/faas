package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	tasksStream     = "TASKS"
	tasksBucket     = "tasks"
	functionsBucket = "functions"
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

type UnifiedStorage struct {
	Conn       *nats.Conn
	JS         jetstream.JetStream
	TaskStream jetstream.Stream
	TaskMeta   jetstream.KeyValue
	FuncObj    jetstream.ObjectStore
	FuncMeta   jetstream.KeyValue
}

func NewUnifiedStorage(url string) (*UnifiedStorage, error) {
	conn, err := NewConnection(url)
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(conn)
	if err != nil {
		return nil, fmt.Errorf("create jetstream: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	taskStream, err := js.Stream(ctx, tasksStream)
	if err != nil {
		return nil, fmt.Errorf("connect to stream %s: %w", tasksStream, err)
	}

	taskMeta, err := js.KeyValue(ctx, tasksBucket)
	if err != nil {
		return nil, fmt.Errorf("connect to kv %s: %w", tasksBucket, err)
	}

	funcMeta, err := js.KeyValue(ctx, functionsBucket)
	if err != nil {
		return nil, fmt.Errorf("connect to kv %s: %w", functionsBucket, err)
	}

	funcObj, err := js.ObjectStore(ctx, functionsBucket)
	if err != nil {
		return nil, fmt.Errorf("connect to obj %s: %w", functionsBucket, err)
	}

	return &UnifiedStorage{
		Conn:       conn,
		JS:         js,
		TaskStream: taskStream,
		TaskMeta:   taskMeta,
		FuncMeta:   funcMeta,
		FuncObj:    funcObj,
	}, nil
}
