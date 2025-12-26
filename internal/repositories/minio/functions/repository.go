package funcrepo

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

type Repository struct {
	objectStorage *minio.Client
	bucketName    string
}

func NewRepository(objectStorage *minio.Client, bucketName string) (*Repository, error) {
	exists, err := objectStorage.BucketExists(context.Background(), bucketName)
	if err != nil {
		return nil, fmt.Errorf("cannot check if functions bucket exists: %w", err)
	}

	if !exists {
		err := objectStorage.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("cannot create functions bucket: %w", err)
		}
	}

	return &Repository{
		objectStorage: objectStorage,
		bucketName:    bucketName,
	}, nil
}

func (r *Repository) GetFunction(ctx context.Context, path string) (io.Reader, error) {
	function, err := r.objectStorage.GetObject(ctx, r.bucketName, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get function %q: %w", path, err)
	}

	if _, err := function.Stat(); err != nil {
		function.Close()
		return nil, fmt.Errorf("function %q not found or inaccessible: %w", path, err)
	}

	return function, nil
}
