package mailwrite

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
		return errors.New("mail write request size or object is invalid")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(value) != nil || d.Decode(new(any)) != io.EOF {
		return errors.New("mail write request contains invalid or undeclared fields")
	}
	return nil
}

func (r *SendRequest) UnmarshalJSON(raw []byte) error {
	type plain SendRequest
	var value plain
	if err := decode(raw, &value); err != nil {
		return err
	}
	*r = SendRequest(value)
	return r.Validate()
}

func (r *ReplyRequest) UnmarshalJSON(raw []byte) error {
	type plain ReplyRequest
	var value plain
	if err := decode(raw, &value); err != nil {
		return err
	}
	*r = ReplyRequest(value)
	return r.Validate()
}
