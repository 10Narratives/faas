package opdomain

import "errors"

var (
	ErrOperationNotFound      error = errors.New("operation not found")
	ErrOperationAlreadyExists error = errors.New("operation already exists")
	ErrFailedPrecondition     error = errors.New("failed precondition")
)
