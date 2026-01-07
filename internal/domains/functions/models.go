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
	Bucket    string `json:"bucket"`
	ObjectKey string `json:"object_key"`
	Size      uint64 `json:"size"`
	SHA256    string `json:"sha_256"`
}

type Function struct {
	InternalID  uuid.UUID     `json:"internal_id"`
	Name        FunctionName  `json:"name"`
	DisplayName string        `json:"display_name"`
	UploadedAt  time.Time     `json:"uploaded_at"`
	Bundle      *SourceBundle `json:"bundle,omitzero"`
}
