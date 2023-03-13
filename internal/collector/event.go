package collector

import (
	"context"
	"encoding/json"
)

type SnowplowCollectorPayload struct {
	Schema string          `json:"schema"`
	Data   json.RawMessage `json:"data"`
}

func BuildSnowplowPayload[T any](ctx context.Context, raw []byte) (*T, error) {
	payload := new(T)

	if err := json.Unmarshal(raw, payload); err != nil {
		return nil, err
	}

	return payload, nil
}
