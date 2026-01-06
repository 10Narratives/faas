package opdomain

import "errors"

var (
	ErrOperationNotFound    = errors.New("operation not found")
	ErrInvalidOperationName = errors.New("invalid operation name")
	ErrInvalidArgument      = errors.New("invalid argument")
)
