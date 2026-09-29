package collaborationwrite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func (request *SendRequest) UnmarshalJSON(raw []byte) error {
	type plain SendRequest
	var value plain
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("collaboration write request has trailing content")
	}
	*request = SendRequest(value)
	return request.Validate()
}
