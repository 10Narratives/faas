package opdomain

import "encoding/json"

type Operation struct {
	Name     string
	Metadata json.RawMessage
	Result   json.RawMessage
	Done     bool
}
