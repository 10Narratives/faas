package taskdomain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TaskState int

const (
	TaskStateUnspecified TaskState = iota
	TaskStatePending
	TaskStateProcessing
	TaskStateSucceeded
	TaskStateFailed
	TaskStateCanceled
)

type TaskName string

func ParseTaskName(s string) (TaskName, error) {
	const prefix = "tasks/"
	if len(s) <= len(prefix) || s[:len(prefix)] != prefix {
		return "", fmt.Errorf("%w: %q", ErrInvalidName, s)
	}
	return TaskName(s), nil
}

type Task struct {
	ID         uuid.UUID   `json:"id"`
	Name       TaskName    `json:"name"`
	Function   string      `json:"function"`
	Parameters string      `json:"parameters"`
	State      TaskState   `json:"state"`
	CreatedAt  time.Time   `json:"created_at"`
	StartedAt  time.Time   `json:"started_at"`
	EndedAt    time.Time   `json:"ended_at"`
	Result     *TaskResult `json:"result,omitempty"`
}

type TaskResultType string

const (
	TaskResultInline    TaskResultType = "inline_result"
	TaskResultObjectKey TaskResultType = "object_key"
	TaskResultError     TaskResultType = "error"
)

type TaskResult struct {
	Type         TaskResultType `json:"type"`
	InlineResult []byte         `json:"inline_result,omitempty"`
	ObjectKey    string         `json:"object_key,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

func NewInlineResult(b []byte) TaskResult {
	return TaskResult{Type: TaskResultInline, InlineResult: b}
}
func NewObjectKey(k string) TaskResult {
	return TaskResult{Type: TaskResultObjectKey, ObjectKey: k}
}
func NewError(msg string) TaskResult {
	return TaskResult{Type: TaskResultError, ErrorMessage: msg}
}

func (tr TaskResult) Validate() error {
	set := 0
	if len(tr.InlineResult) > 0 {
		set++
	}
	if tr.ObjectKey != "" {
		set++
	}
	if tr.ErrorMessage != "" {
		set++
	}
	if set != 1 {
		return fmt.Errorf("invalid TaskResult: expected exactly 1 field set, got %d", set)
	}

	switch tr.Type {
	case TaskResultInline:
		if len(tr.InlineResult) == 0 {
			return fmt.Errorf("inline_result is empty")
		}
	case TaskResultObjectKey:
		if tr.ObjectKey == "" {
			return fmt.Errorf("object_key is empty")
		}
	case TaskResultError:
		if tr.ErrorMessage == "" {
			return fmt.Errorf("error_message is empty")
		}
	default:
		return fmt.Errorf("unknown kind: %q", tr.Type)
	}
	return nil
}
