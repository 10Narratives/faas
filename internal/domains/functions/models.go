package funcdomain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type FunctionName string

func ParseFunctionName(s string) (FunctionName, error) {
	const prefix = "functions/"
	if len(s) <= len(prefix) || s[:len(prefix)] != prefix {
		return "", fmt.Errorf("%w: %q", ErrInvalidName, s)
	}
	return FunctionName(s), nil
}

type UploadFunctionFormat string

const (
	ZipFormat   UploadFunctionFormat = "zip"
	TarGZFormat UploadFunctionFormat = "tar.gz"
)

type SourceBundle struct {
	Bucket    string
	ObjectKey string
	Size      uint64
	SHA256    string
}

type Function struct {
	InternalID  uuid.UUID
	Name        FunctionName
	DisplayName string
	UploadedAt  time.Time
	Bundle      *SourceBundle
}
