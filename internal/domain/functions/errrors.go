package funcdomain

import "errors"

var (
	ErrFunctionNotFound      error = errors.New("function not found")
	ErrFunctionAlreadyExists error = errors.New("function already exists")
)
