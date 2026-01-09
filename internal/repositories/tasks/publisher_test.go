package taskrepo_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	taskdomain "github.com/10Narratives/faas/internal/domains/tasks"
	taskrepo "github.com/10Narratives/faas/internal/repositories/tasks"
	"github.com/10Narratives/faas/internal/repositories/tasks/mocks"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPublisher_PublishExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("error: nil msg", func(t *testing.T) {
		js := mocks.NewStream(t)
		p := taskrepo.NewPublisher(js)

		err := p.PublishExecute(ctx, nil)
		require.Error(t, err)
		require.Equal(t, "execute message is nil", err.Error())
	})

	t.Run("error: empty task name", func(t *testing.T) {
		js := mocks.NewStream(t)
		p := taskrepo.NewPublisher(js)

		err := p.PublishExecute(ctx, &taskdomain.ExecuteTaskMessage{})
		require.ErrorIs(t, err, taskdomain.ErrInvalidName)
	})

	t.Run("error: js publish fails", func(t *testing.T) {
		js := mocks.NewStream(t)
		p := taskrepo.NewPublisher(js)

		msg := &taskdomain.ExecuteTaskMessage{TaskName: "tasks/123"}

		wantBytes, err := json.Marshal(msg)
		require.NoError(t, err)

		js.EXPECT().
			Publish(ctx, "tasks.execute",
				mock.MatchedBy(func(b []byte) bool { return string(b) == string(wantBytes) }),
			).
			Return((*jetstream.PubAck)(nil), errors.New("nats down")).
			Once()

		err = p.PublishExecute(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "jetstream publish execute:")
	})

	t.Run("ok: publish called with correct subject and payload", func(t *testing.T) {
		js := mocks.NewStream(t)
		p := taskrepo.NewPublisher(js)

		msg := &taskdomain.ExecuteTaskMessage{TaskName: "tasks/123"}

		wantBytes, err := json.Marshal(msg)
		require.NoError(t, err)

		js.EXPECT().
			Publish(ctx, "tasks.execute",
				mock.MatchedBy(func(b []byte) bool { return string(b) == string(wantBytes) }),
			).
			Return(&jetstream.PubAck{}, nil).
			Once()

		err = p.PublishExecute(ctx, msg)
		require.NoError(t, err)
	})
}

func TestPublisher_PublishCancel(t *testing.T) {
	ctx := context.Background()

	t.Run("error: nil msg", func(t *testing.T) {
		js := mocks.NewStream(t)
		p := taskrepo.NewPublisher(js)

		err := p.PublishCancel(ctx, nil)
		require.Error(t, err)
		require.Equal(t, "cancel message is nil", err.Error())
	})

	t.Run("error: empty task name", func(t *testing.T) {
		js := mocks.NewStream(t)
		p := taskrepo.NewPublisher(js)

		err := p.PublishCancel(ctx, &taskdomain.CancelTaskMessage{})
		require.ErrorIs(t, err, taskdomain.ErrInvalidName)
	})

	t.Run("error: js publish fails", func(t *testing.T) {
		js := mocks.NewStream(t)
		p := taskrepo.NewPublisher(js)

		msg := &taskdomain.CancelTaskMessage{TaskName: "tasks/123"}

		wantBytes, err := json.Marshal(msg)
		require.NoError(t, err)

		js.EXPECT().
			Publish(ctx, "tasks.cancel",
				mock.MatchedBy(func(b []byte) bool { return string(b) == string(wantBytes) }),
			).
			Return((*jetstream.PubAck)(nil), errors.New("nats down")).
			Once()

		err = p.PublishCancel(ctx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "jetstream publish cancel:")
	})

	t.Run("ok: publish called with correct subject and payload", func(t *testing.T) {
		js := mocks.NewStream(t)
		p := taskrepo.NewPublisher(js)

		msg := &taskdomain.CancelTaskMessage{TaskName: "tasks/123"}

		wantBytes, err := json.Marshal(msg)
		require.NoError(t, err)

		js.EXPECT().
			Publish(ctx, "tasks.cancel",
				mock.MatchedBy(func(b []byte) bool { return string(b) == string(wantBytes) }),
			).
			Return(&jetstream.PubAck{}, nil).
			Once()

		err = p.PublishCancel(ctx, msg)
		require.NoError(t, err)
	})
}
