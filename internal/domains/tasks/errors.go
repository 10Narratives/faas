package taskdomain

import "errors"

var (
	ErrInvalidName          = errors.New("invalid task name")
	ErrNotFound             = errors.New("task not found")
	ErrAlreadyExists        = errors.New("task already exists")
	ErrInvalidState         = errors.New("invalid task state")
	ErrInvalidParameters    = errors.New("invalid task parameters")
	ErrInvalidFunction      = errors.New("invalid function name")
	ErrTaskNotPending       = errors.New("task is not in pending state")
	ErrTaskNotProcessing    = errors.New("task is not in processing state")
	ErrTaskAlreadyCompleted = errors.New("task already completed")
	ErrCannotCancelTask     = errors.New("cannot cancel task in current state")
	ErrEmptyPageSize        = errors.New("page size must be greater than 0")
	ErrInvalidPageToken     = errors.New("invalid page token")
	ErrResultAlreadySet     = errors.New("task result already set")
	ErrInvalidResult        = errors.New("invalid task result")
	ErrUnknownResultType    = errors.New("unknown result type")
)
