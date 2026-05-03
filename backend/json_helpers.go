package main

import (
	"bytes"
	"encoding/json"
)

// jsonDecoder is the project's strict JSON decoder factory. Centralizing here
// lets us tighten policy in one place (e.g. UseNumber, DisallowUnknownFields)
// without grepping every handler.
type jsonDecoder struct {
	d *json.Decoder
}

func newJSONDecoder(body []byte) *jsonDecoder {
	return &jsonDecoder{d: json.NewDecoder(bytes.NewReader(body))}
}

func (j *jsonDecoder) DisallowUnknownFields() { j.d.DisallowUnknownFields() }

func (j *jsonDecoder) Decode(v any) error { return j.d.Decode(v) }
