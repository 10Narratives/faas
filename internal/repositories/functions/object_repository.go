package funcrepo

import (
	"context"
	"io"
	"strings"

	funcdomain "github.com/10Narratives/faas/internal/domains/functions"
	"github.com/nats-io/nats.go/jetstream"
)

type ObjectRepository struct {
	bucket string
	os     jetstream.ObjectStore
}

func NewObjectRepository(ctx context.Context, js jetstream.JetStream, bucket string) (*ObjectRepository, error) {
	os, err := js.ObjectStore(ctx, bucket)
	if err != nil {
		os, err = js.CreateObjectStore(ctx, jetstream.ObjectStoreConfig{Bucket: bucket})
		if err != nil {
			return nil, err
		}
	}
	return &ObjectRepository{bucket: bucket, os: os}, nil
}

func (r *ObjectRepository) SaveBundle(
	ctx context.Context,
	name funcdomain.FunctionName,
	format funcdomain.UploadFunctionFormat,
	data io.ReadCloser,
) (*funcdomain.SourceBundle, error) {
	if data == nil {
		return nil, funcdomain.ErrInvalidArgument
	}
	defer data.Close()

	key := objectKey(name, format)

	if _, err := r.os.GetInfo(ctx, key); err == nil {
		return nil, funcdomain.ErrFunctionAlreadyExists
	} else if !isObjectNotFound(err) {
		return nil, err
	}

	info, err := r.os.Put(ctx, jetstream.ObjectMeta{Name: key}, data)
	if err != nil {
		return nil, err
	}

	return &funcdomain.SourceBundle{
		Bucket:    r.bucket,
		ObjectKey: key,
		Size:      info.Size,
		SHA256:    info.Digest,
	}, nil
}

func (r *ObjectRepository) OpenBundle(ctx context.Context, bundle *funcdomain.SourceBundle) (io.ReadCloser, error) {
	if bundle == nil || bundle.ObjectKey == "" {
		return nil, funcdomain.ErrInvalidArgument
	}
	return r.os.Get(ctx, bundle.ObjectKey)
}

func (r *ObjectRepository) DeleteBundle(ctx context.Context, bundle *funcdomain.SourceBundle) error {
	if bundle == nil || bundle.ObjectKey == "" {
		return funcdomain.ErrInvalidArgument
	}

	return r.os.Delete(ctx, bundle.ObjectKey)
}

func objectKey(name funcdomain.FunctionName, format funcdomain.UploadFunctionFormat) string {
	s := strings.TrimPrefix(string(name), "functions/")
	s = strings.ReplaceAll(s, "/", "_")
	return s + "." + string(format)
}

func isObjectNotFound(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(strings.ToLower(err.Error()), "object not found") ||
		strings.Contains(strings.ToLower(err.Error()), "not found")
}
