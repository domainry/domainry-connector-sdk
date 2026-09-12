package calendarwrite

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// JSON escaping can expand each valid text byte to six bytes.
const MaximumRequestBytes = 1 << 20

func decode(raw []byte, value any) error {
	if len(raw) == 0 || len(raw) > MaximumRequestBytes || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return errors.New("calendar write request size or object is invalid")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(value) != nil || d.Decode(new(any)) != io.EOF {
		return errors.New("calendar write request contains invalid or undeclared fields")
	}
	return nil
}

func (r *CreateRequest) UnmarshalJSON(raw []byte) error {
	type plain CreateRequest
	var value plain
	if err := decode(raw, &value); err != nil {
		return err
	}
	*r = CreateRequest(value)
	return r.Validate()
}

func (r *UpdateRequest) UnmarshalJSON(raw []byte) error {
	type plain UpdateRequest
	var value plain
	if err := decode(raw, &value); err != nil {
		return err
	}
	*r = UpdateRequest(value)
	return r.Validate()
}

func (r *InspectRequest) UnmarshalJSON(raw []byte) error {
	type plain InspectRequest
	var value plain
	if err := decode(raw, &value); err != nil {
		return err
	}
	*r = InspectRequest(value)
	return r.Validate()
}
