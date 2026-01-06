package opdomain

import "fmt"

type OperationName string

func ParseOperationName(s string) (OperationName, error) {
	if len(s) == 0 || !containsOperationsSegment(s) {
		return "", fmt.Errorf("%w: %q", ErrInvalidOperationName, s)
	}
	return OperationName(s), nil
}

func containsOperationsSegment(s string) bool {
	for i := 0; i+11 <= len(s); i++ {
		if s[i:i+11] == "operations/" {
			return true
		}
	}
	return false
}

type Any struct {
	TypeURL string
	Value   []byte
}

type Status struct {
	Code    int32
	Message string
	Details []*Any
}

type Operation struct {
	Name OperationName

	Done bool

	Metadata *Any
	Response *Any
	Error    *Status
}
