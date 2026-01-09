package funcrepo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"time"

	funcdomain "github.com/10Narratives/faas/internal/domains/functions"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
)

type MetadataRepository struct {
	kv jetstream.KeyValue
}

func NewMetadataRepository(kv jetstream.KeyValue) *MetadataRepository {
	return &MetadataRepository{kv: kv}
}

func (r *MetadataRepository) CreateFunction(ctx context.Context, fn *funcdomain.Function) error {
	if fn == nil || fn.Name == "" || fn.Bundle == nil {
		return funcdomain.ErrInvalidArgument
	}

	key := keyFromFunctionName(fn.Name)

	b, err := json.Marshal(toStored(fn))
	if err != nil {
		return err
	}

	_, err = r.kv.Create(ctx, key, b)
	if err != nil {
		if isKVKeyExists(err) {
			return funcdomain.ErrFunctionAlreadyExists
		}
		return err
	}
	return nil
}

func (r *MetadataRepository) GetFunction(ctx context.Context, args *funcdomain.GetFunctionArgs) (*funcdomain.GetFunctionResult, error) {
	if args == nil || args.Name == "" {
		return nil, funcdomain.ErrInvalidArgument
	}

	key := keyFromFunctionName(args.Name)

	e, err := r.kv.Get(ctx, key)
	if err != nil {
		if isKVKeyNotFound(err) {
			return nil, funcdomain.ErrFunctionNotFound
		}
		return nil, err
	}

	var sf storedFunction
	if err := json.Unmarshal(e.Value(), &sf); err != nil {
		return nil, err
	}

	fn, err := fromStored(&sf)
	if err != nil {
		return nil, err
	}
	return &funcdomain.GetFunctionResult{Function: fn}, nil
}

func (r *MetadataRepository) DeleteFunction(ctx context.Context, args *funcdomain.DeleteFunctionArgs) error {
	if args == nil || args.Name == "" {
		return funcdomain.ErrInvalidArgument
	}
	// ВАЖНО: в вашем funcsrv.DeleteFunction() перед удалением уже делается GetFunction(),
	// поэтому повторная проверка существования тут не нужна.
	return r.kv.Delete(ctx, keyFromFunctionName(args.Name))
}

func (r *MetadataRepository) ListFunctions(ctx context.Context, args *funcdomain.ListFunctionsArgs) (*funcdomain.ListFunctionsResult, error) {
	if args == nil {
		return nil, funcdomain.ErrInvalidArgument
	}

	pageSize := int(args.PageSize)
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	keysLister, err := r.kv.ListKeys(ctx)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, 128)
	for k := range keysLister.Keys() {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	start, err := startIndexFromToken(keys, args.PageToken)
	if err != nil {
		return nil, err
	}
	if start >= len(keys) {
		return &funcdomain.ListFunctionsResult{Functions: nil, NextPageToken: ""}, nil
	}

	end := start + pageSize
	if end > len(keys) {
		end = len(keys)
	}

	out := make([]*funcdomain.Function, 0, end-start)
	for _, k := range keys[start:end] {
		e, err := r.kv.Get(ctx, k)
		if err != nil {
			if isKVKeyNotFound(err) {
				continue
			}
			return nil, err
		}

		var sf storedFunction
		if err := json.Unmarshal(e.Value(), &sf); err != nil {
			return nil, err
		}
		fn, err := fromStored(&sf)
		if err != nil {
			return nil, err
		}
		out = append(out, fn)
	}

	nextToken := ""
	if end < len(keys) {
		nextToken = keys[end-1] // PageToken = “последний key на странице”
	}

	return &funcdomain.ListFunctionsResult{Functions: out, NextPageToken: nextToken}, nil
}

// --- storage format ---

type storedFunction struct {
	InternalID  string                   `json:"internal_id"`
	Name        string                   `json:"name"`
	DisplayName string                   `json:"display_name"`
	UploadedAt  time.Time                `json:"uploaded_at"`
	Bundle      *funcdomain.SourceBundle `json:"bundle"`
}

func toStored(fn *funcdomain.Function) *storedFunction {
	return &storedFunction{
		InternalID:  fn.InternalID.String(),
		Name:        string(fn.Name),
		DisplayName: fn.DisplayName,
		UploadedAt:  fn.UploadedAt,
		Bundle:      fn.Bundle,
	}
}

func fromStored(sf *storedFunction) (*funcdomain.Function, error) {
	if sf == nil || sf.Name == "" || sf.Bundle == nil {
		return nil, funcdomain.ErrInvalidArgument
	}
	id, err := uuid.Parse(sf.InternalID)
	if err != nil {
		return nil, err
	}
	name, err := funcdomain.ParseFunctionName(sf.Name)
	if err != nil {
		return nil, err
	}
	return &funcdomain.Function{
		InternalID:  id,
		Name:        name,
		DisplayName: sf.DisplayName,
		UploadedAt:  sf.UploadedAt,
		Bundle:      sf.Bundle,
	}, nil
}

// --- key + paging ---

func keyFromFunctionName(name funcdomain.FunctionName) string {
	// KV key должен быть subject-like, поэтому кодируем имя в base64url и кладём в один token. [web:61]
	enc := base64.RawURLEncoding.EncodeToString([]byte(name))
	return "fn." + enc
}

// PageToken хранит “последний key предыдущей страницы”.
func startIndexFromToken(sortedKeys []string, lastKey string) (int, error) {
	if lastKey == "" {
		return 0, nil
	}
	i := sort.SearchStrings(sortedKeys, lastKey)
	if i >= len(sortedKeys) || sortedKeys[i] != lastKey {
		return 0, funcdomain.ErrInvalidPageToken
	}
	return i + 1, nil
}

// --- error helpers ---

func isKVKeyNotFound(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	// В примерах nats.go это “nats: key not found”. [web:44]
	return strings.Contains(s, "key not found") || strings.Contains(s, "not found")
}

func isKVKeyExists(err error) bool {
	if err == nil {
		return false
	}
	// В примерах nats.go это “nats: key exists”. [web:44]
	return strings.Contains(strings.ToLower(err.Error()), "key exists")
}
