package collector

import (
	"context"
	"encoding/json"
)

type SnowplowCollectorPayload struct {
	Schema string          `json:"schema"`
	Data   json.RawMessage `json:"data"`
}

// BuildSnowplowPayload parses a JSON payload into a generic type T.
//
// ctx - The context for the operation. Currently unused.
// raw - The raw JSON payload bytes to decode.
//
// Returns a pointer to a T instance containing the decoded payload, or an error
// if the JSON could not be decoded.
func BuildSnowplowPayload[T any](ctx context.Context, raw []byte) (*T, error) {
	payload := new(T)

	if err := json.Unmarshal(raw, payload); err != nil {
		return nil, err
	}

	return payload, nil
}
