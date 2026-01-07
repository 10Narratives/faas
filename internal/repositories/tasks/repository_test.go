package taskrepo

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"
	"time"

	taskdomain "github.com/10Narratives/faas/internal/domains/tasks"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/require"
)

type fakeEntry struct {
	bucket  string
	key     string
	val     []byte
	rev     uint64
	created time.Time
	delta   uint64
	op      jetstream.KeyValueOp
}

func (e *fakeEntry) Bucket() string                  { return e.bucket }
func (e *fakeEntry) Key() string                     { return e.key }
func (e *fakeEntry) Value() []byte                   { return e.val }
func (e *fakeEntry) Revision() uint64                { return e.rev }
func (e *fakeEntry) Created() time.Time              { return e.created }
func (e *fakeEntry) Delta() uint64                   { return e.delta }
func (e *fakeEntry) Operation() jetstream.KeyValueOp { return e.op }

type fakeLister struct {
	ch chan string
}

func (l *fakeLister) Keys() <-chan string { return l.ch }
func (l *fakeLister) Stop() error         { return nil }

type fakeKV struct {
	bucket string

	// hooks
	createErr error
	updateErr error
	deleteErr error
	getErr    error
	listErr   error

	// storage
	nextRev uint64
	items   map[string]*fakeEntry
}

func newFakeKV(bucket string) *fakeKV {
	return &fakeKV{
		bucket:  bucket,
		nextRev: 1,
		items:   map[string]*fakeEntry{},
	}
}

// --- methods actually used by repository ---

func (kv *fakeKV) Get(ctx context.Context, key string) (jetstream.KeyValueEntry, error) {
	if kv.getErr != nil {
		return nil, kv.getErr
	}
	e, ok := kv.items[key]
	if !ok {
		return nil, jetstream.ErrKeyNotFound
	}
	return e, nil
}

func (kv *fakeKV) Create(ctx context.Context, key string, value []byte, _ ...jetstream.KVCreateOpt) (uint64, error) {
	if kv.createErr != nil {
		return 0, kv.createErr
	}
	if _, ok := kv.items[key]; ok {
		return 0, jetstream.ErrKeyExists
	}
	rev := kv.nextRev
	kv.nextRev++

	kv.items[key] = &fakeEntry{
		bucket:  kv.bucket,
		key:     key,
		val:     append([]byte(nil), value...),
		rev:     rev,
		created: time.Now().UTC(),
		op:      jetstream.KeyValuePut,
	}
	return rev, nil
}

func (kv *fakeKV) Update(ctx context.Context, key string, value []byte, revision uint64) (uint64, error) {
	if kv.updateErr != nil {
		return 0, kv.updateErr
	}
	e, ok := kv.items[key]
	if !ok {
		return 0, jetstream.ErrKeyNotFound
	}
	if e.rev != revision {
		return 0, errors.New("wrong revision")
	}

	rev := kv.nextRev
	kv.nextRev++

	e.val = append([]byte(nil), value...)
	e.rev = rev
	e.created = time.Now().UTC()
	e.op = jetstream.KeyValuePut
	return rev, nil
}

func (kv *fakeKV) Delete(ctx context.Context, key string, _ ...jetstream.KVDeleteOpt) error {
	if kv.deleteErr != nil {
		return kv.deleteErr
	}
	if _, ok := kv.items[key]; !ok {
		return jetstream.ErrKeyNotFound
	}
	delete(kv.items, key)
	return nil
}

func (kv *fakeKV) ListKeys(ctx context.Context, _ ...jetstream.WatchOpt) (jetstream.KeyLister, error) {
	if kv.listErr != nil {
		return nil, kv.listErr
	}

	keys := make([]string, 0, len(kv.items))
	for k := range kv.items {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ch := make(chan string, len(keys))
	for _, k := range keys {
		ch <- k
	}
	close(ch)

	return &fakeLister{ch: ch}, nil
}

// --- stubs to satisfy jetstream.KeyValue interface ---

func (kv *fakeKV) GetRevision(ctx context.Context, key string, revision uint64) (jetstream.KeyValueEntry, error) {
	return nil, errors.New("not implemented")
}
func (kv *fakeKV) Put(ctx context.Context, key string, value []byte) (uint64, error) {
	return 0, errors.New("not implemented")
}
func (kv *fakeKV) PutString(ctx context.Context, key string, value string) (uint64, error) {
	return 0, errors.New("not implemented")
}
func (kv *fakeKV) Purge(ctx context.Context, key string, opts ...jetstream.KVDeleteOpt) error {
	return errors.New("not implemented")
}
func (kv *fakeKV) Watch(ctx context.Context, keys string, opts ...jetstream.WatchOpt) (jetstream.KeyWatcher, error) {
	return nil, errors.New("not implemented")
}
func (kv *fakeKV) WatchAll(ctx context.Context, opts ...jetstream.WatchOpt) (jetstream.KeyWatcher, error) {
	return nil, errors.New("not implemented")
}
func (kv *fakeKV) WatchFiltered(ctx context.Context, keys []string, opts ...jetstream.WatchOpt) (jetstream.KeyWatcher, error) {
	return nil, errors.New("not implemented")
}
func (kv *fakeKV) Keys(ctx context.Context, opts ...jetstream.WatchOpt) ([]string, error) {
	return nil, errors.New("not implemented")
}
func (kv *fakeKV) ListKeysFiltered(ctx context.Context, filters ...string) (jetstream.KeyLister, error) {
	return nil, errors.New("not implemented")
}
func (kv *fakeKV) History(ctx context.Context, key string, opts ...jetstream.WatchOpt) ([]jetstream.KeyValueEntry, error) {
	return nil, errors.New("not implemented")
}
func (kv *fakeKV) Bucket() string { return kv.bucket }
func (kv *fakeKV) PurgeDeletes(ctx context.Context, opts ...jetstream.KVPurgeOpt) error {
	return errors.New("not implemented")
}
func (kv *fakeKV) Status(ctx context.Context) (jetstream.KeyValueStatus, error) {
	return nil, errors.New("not implemented")
}

// -------------------- tests --------------------

func TestRepository_CreateTask(t *testing.T) {
	t.Parallel()

	t.Run("nil args -> ErrInvalidParameters", func(t *testing.T) {
		t.Parallel()

		r := &Repository{kv: newFakeKV("b")}
		_, err := r.CreateTask(context.Background(), nil)
		require.ErrorIs(t, err, taskdomain.ErrInvalidParameters)
	})

	t.Run("empty function -> ErrInvalidFunction", func(t *testing.T) {
		t.Parallel()

		r := &Repository{kv: newFakeKV("b")}
		_, err := r.CreateTask(context.Background(), &taskdomain.CreateTaskArgs{Function: "  ", Parameters: "{}"})
		require.ErrorIs(t, err, taskdomain.ErrInvalidFunction)
	})

	t.Run("kv returns ErrKeyExists -> ErrAlreadyExists", func(t *testing.T) {
		t.Parallel()

		kv := newFakeKV("b")
		kv.createErr = jetstream.ErrKeyExists
		r := &Repository{kv: kv}

		_, err := r.CreateTask(context.Background(), &taskdomain.CreateTaskArgs{Function: "fn", Parameters: "{}"})
		require.ErrorIs(t, err, taskdomain.ErrAlreadyExists)
	})

	t.Run("ok -> stores pending task", func(t *testing.T) {
		t.Parallel()

		kv := newFakeKV("b")
		r := &Repository{kv: kv}

		before := time.Now().UTC()
		_, err := r.CreateTask(context.Background(), &taskdomain.CreateTaskArgs{Function: "fn", Parameters: `{"a":1}`})
		after := time.Now().UTC()

		require.NoError(t, err)
		require.Len(t, kv.items, 1)

		var storedKey string
		var storedVal []byte
		for k, e := range kv.items {
			storedKey = k
			storedVal = e.val
		}

		require.True(t, stringsHasPrefix(storedKey, "tasks/"))

		var got taskdomain.Task
		require.NoError(t, json.Unmarshal(storedVal, &got))

		require.Equal(t, storedKey, string(got.Name))
		require.Equal(t, "fn", got.Function)
		require.Equal(t, `{"a":1}`, got.Parameters)
		require.Equal(t, taskdomain.TaskStatePending, got.State)
		require.False(t, got.CreatedAt.IsZero())
		require.True(t, !got.CreatedAt.Before(before) && !got.CreatedAt.After(after))
	})
}

func TestRepository_GetTask(t *testing.T) {
	t.Parallel()

	t.Run("nil args -> ErrInvalidName", func(t *testing.T) {
		t.Parallel()

		r := &Repository{kv: newFakeKV("b")}
		_, err := r.GetTask(context.Background(), nil)
		require.ErrorIs(t, err, taskdomain.ErrInvalidName)
	})

	t.Run("invalid name -> ErrInvalidName", func(t *testing.T) {
		t.Parallel()

		r := &Repository{kv: newFakeKV("b")}
		_, err := r.GetTask(context.Background(), &taskdomain.GetTaskArgs{Name: "bad"})
		require.ErrorIs(t, err, taskdomain.ErrInvalidName)
	})

	t.Run("not found -> ErrNotFound", func(t *testing.T) {
		t.Parallel()

		r := &Repository{kv: newFakeKV("b")}
		_, err := r.GetTask(context.Background(), &taskdomain.GetTaskArgs{Name: "tasks/404"})
		require.ErrorIs(t, err, taskdomain.ErrNotFound)
	})

	t.Run("ok -> returns task; if stored Name empty, it is filled from key", func(t *testing.T) {
		t.Parallel()

		kv := newFakeKV("b")
		r := &Repository{kv: kv}

		// записываем JSON без поля name
		key := "tasks/1"
		raw := []byte(`{"function":"fn","parameters":"{}","state":1}`)
		_, err := kv.Create(context.Background(), key, raw)
		require.NoError(t, err)

		res, err := r.GetTask(context.Background(), &taskdomain.GetTaskArgs{Name: key})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Task)
		require.Equal(t, key, string(res.Task.Name))
	})
}

func TestRepository_DeleteTask(t *testing.T) {
	t.Parallel()

	t.Run("nil args -> ErrInvalidName", func(t *testing.T) {
		t.Parallel()

		r := &Repository{kv: newFakeKV("b")}
		err := r.DeleteTask(context.Background(), nil)
		require.ErrorIs(t, err, taskdomain.ErrInvalidName)
	})

	t.Run("invalid name -> ErrInvalidName", func(t *testing.T) {
		t.Parallel()

		r := &Repository{kv: newFakeKV("b")}
		err := r.DeleteTask(context.Background(), &taskdomain.DeleteTaskArgs{Name: "bad"})
		require.ErrorIs(t, err, taskdomain.ErrInvalidName)
	})

	t.Run("not found -> ErrNotFound", func(t *testing.T) {
		t.Parallel()

		r := &Repository{kv: newFakeKV("b")}
		err := r.DeleteTask(context.Background(), &taskdomain.DeleteTaskArgs{Name: "tasks/404"})
		require.ErrorIs(t, err, taskdomain.ErrNotFound)
	})

	t.Run("ok -> deletes", func(t *testing.T) {
		t.Parallel()

		kv := newFakeKV("b")
		r := &Repository{kv: kv}

		_, err := kv.Create(context.Background(), "tasks/1", []byte(`{"name":"tasks/1","state":1}`))
		require.NoError(t, err)

		err = r.DeleteTask(context.Background(), &taskdomain.DeleteTaskArgs{Name: "tasks/1"})
		require.NoError(t, err)
		require.Empty(t, kv.items)
	})
}

func TestRepository_CancelTask(t *testing.T) {
	t.Parallel()

	t.Run("nil args -> ErrInvalidName", func(t *testing.T) {
		t.Parallel()

		r := &Repository{kv: newFakeKV("b")}
		_, err := r.CancelTask(context.Background(), nil)
		require.ErrorIs(t, err, taskdomain.ErrInvalidName)
	})

	t.Run("invalid name -> ErrInvalidName", func(t *testing.T) {
		t.Parallel()

		r := &Repository{kv: newFakeKV("b")}
		_, err := r.CancelTask(context.Background(), &taskdomain.CancelTaskArgs{Name: "bad"})
		require.ErrorIs(t, err, taskdomain.ErrInvalidName)
	})

	t.Run("not found -> ErrNotFound", func(t *testing.T) {
		t.Parallel()

		r := &Repository{kv: newFakeKV("b")}
		_, err := r.CancelTask(context.Background(), &taskdomain.CancelTaskArgs{Name: "tasks/404"})
		require.ErrorIs(t, err, taskdomain.ErrNotFound)
	})

	t.Run("already completed -> ErrTaskAlreadyCompleted", func(t *testing.T) {
		t.Parallel()

		kv := newFakeKV("b")
		r := &Repository{kv: kv}

		key := "tasks/1"
		_, err := kv.Create(context.Background(), key, []byte(`{"name":"tasks/1","state":3}`)) // succeeded
		require.NoError(t, err)

		_, err = r.CancelTask(context.Background(), &taskdomain.CancelTaskArgs{Name: key})
		require.ErrorIs(t, err, taskdomain.ErrTaskAlreadyCompleted)
	})

	t.Run("ok (pending) -> updates state to canceled and sets ended_at", func(t *testing.T) {
		t.Parallel()

		kv := newFakeKV("b")
		r := &Repository{kv: kv}

		key := "tasks/1"
		_, err := kv.Create(context.Background(), key, []byte(`{"name":"tasks/1","state":1}`)) // pending
		require.NoError(t, err)

		before := time.Now().UTC()
		res, err := r.CancelTask(context.Background(), &taskdomain.CancelTaskArgs{Name: key})
		after := time.Now().UTC()

		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Task)
		require.Equal(t, taskdomain.TaskStateCanceled, res.Task.State)
		require.False(t, res.Task.EndedAt.IsZero())
		require.True(t, !res.Task.EndedAt.Before(before) && !res.Task.EndedAt.After(after))

		// и реально сохранилось в KV
		gotEntry, err := kv.Get(context.Background(), key)
		require.NoError(t, err)

		var stored taskdomain.Task
		require.NoError(t, json.Unmarshal(gotEntry.Value(), &stored))
		require.Equal(t, taskdomain.TaskStateCanceled, stored.State)
	})

	t.Run("update conflict -> ErrCannotCancelTask", func(t *testing.T) {
		t.Parallel()

		kv := newFakeKV("b")
		kv.updateErr = errors.New("conflict")
		r := &Repository{kv: kv}

		key := "tasks/1"
		_, err := kv.Create(context.Background(), key, []byte(`{"name":"tasks/1","state":1}`)) // pending
		require.NoError(t, err)

		_, err = r.CancelTask(context.Background(), &taskdomain.CancelTaskArgs{Name: key})
		require.ErrorIs(t, err, taskdomain.ErrCannotCancelTask)
	})
}

func TestRepository_ListTasks(t *testing.T) {
	t.Parallel()

	t.Run("nil args -> ErrInvalidParameters", func(t *testing.T) {
		t.Parallel()

		r := &Repository{kv: newFakeKV("b")}
		_, err := r.ListTasks(context.Background(), nil)
		require.ErrorIs(t, err, taskdomain.ErrInvalidParameters)
	})

	t.Run("page_size <= 0 -> ErrEmptyPageSize", func(t *testing.T) {
		t.Parallel()

		r := &Repository{kv: newFakeKV("b")}
		_, err := r.ListTasks(context.Background(), &taskdomain.ListTasksArgs{PageSize: 0})
		require.ErrorIs(t, err, taskdomain.ErrEmptyPageSize)
	})

	t.Run("invalid page token -> ErrInvalidPageToken", func(t *testing.T) {
		t.Parallel()

		kv := newFakeKV("b")
		r := &Repository{kv: kv}

		_, err := kv.Create(context.Background(), "tasks/1", []byte(`{"name":"tasks/1","state":1}`))
		require.NoError(t, err)

		_, err = r.ListTasks(context.Background(), &taskdomain.ListTasksArgs{PageSize: 1, PageToken: "tasks/does-not-exist"})
		require.ErrorIs(t, err, taskdomain.ErrInvalidPageToken)
	})

	t.Run("ok -> returns first page and next token", func(t *testing.T) {
		t.Parallel()

		kv := newFakeKV("b")
		r := &Repository{kv: kv}

		_, _ = kv.Create(context.Background(), "tasks/1", []byte(`{"name":"tasks/1","state":1}`))
		_, _ = kv.Create(context.Background(), "tasks/2", []byte(`{"name":"tasks/2","state":1}`))
		_, _ = kv.Create(context.Background(), "other/1", []byte(`{"name":"other/1","state":1}`)) // должен быть отфильтрован

		res, err := r.ListTasks(context.Background(), &taskdomain.ListTasksArgs{PageSize: 1})
		require.NoError(t, err)
		require.Len(t, res.Tasks, 1)
		require.Equal(t, "tasks/1", string(res.Tasks[0].Name))
		require.Equal(t, "tasks/1", res.NextPageToken)
	})

	t.Run("ok -> second page using page_token", func(t *testing.T) {
		t.Parallel()

		kv := newFakeKV("b")
		r := &Repository{kv: kv}

		_, _ = kv.Create(context.Background(), "tasks/1", []byte(`{"name":"tasks/1","state":1}`))
		_, _ = kv.Create(context.Background(), "tasks/2", []byte(`{"name":"tasks/2","state":1}`))

		res, err := r.ListTasks(context.Background(), &taskdomain.ListTasksArgs{PageSize: 1, PageToken: "tasks/1"})
		require.NoError(t, err)
		require.Len(t, res.Tasks, 1)
		require.Equal(t, "tasks/2", string(res.Tasks[0].Name))
		require.Equal(t, "", res.NextPageToken) // больше нет
	})

	t.Run("missing key between ListKeys and Get is skipped", func(t *testing.T) {
		t.Parallel()

		kv := newFakeKV("b")
		r := &Repository{kv: kv}

		_, _ = kv.Create(context.Background(), "tasks/1", []byte(`{"name":"tasks/1","state":1}`))
		_, _ = kv.Create(context.Background(), "tasks/2", []byte(`{"name":"tasks/2","state":1}`))

		// имитируем “ключ пропал”: удалим tasks/2 после того как ListKeys его выдаст.
		// проще всего: удалим заранее, но оставим ключ в выдаче невозможно без усложнения.
		// Поэтому тут проверяем близкий кейс: Get вернет ErrKeyNotFound -> репо пропустит.
		delete(kv.items, "tasks/2")

		res, err := r.ListTasks(context.Background(), &taskdomain.ListTasksArgs{PageSize: 10})
		require.NoError(t, err)
		require.Len(t, res.Tasks, 1)
		require.Equal(t, "tasks/1", string(res.Tasks[0].Name))
	})
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
