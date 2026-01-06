package funcdomain

import "errors"

var (
	ErrFunctionNotFound      = errors.New("function not found")
	ErrFunctionAlreadyExists = errors.New("function already exists")
	ErrInvalidArgument       = errors.New("invalid argument")
	ErrInvalidName           = errors.New("invalid function name")
	ErrInvalidPageToken      = errors.New("invalid page token")
	ErrUnsupportedFormat     = errors.New("unsupported upload format")
)
