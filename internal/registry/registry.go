package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/qri-io/jsonschema"
	"github.com/wolfeidau/snowdrift/schemas"
)

type SchemaStore struct {
	cache map[string]*jsonschema.Schema
}

func NewSchemaStore() *SchemaStore {
	return &SchemaStore{cache: make(map[string]*jsonschema.Schema)}
}

// Validate validates a JSON payload against a schema.
//
// ctx - The context for the operation. Currently unused.
// schema - The schema ID (e.g. "iglu:com.snowplowanalytics.snowplow/unstruct_event/jsonschema/1-0-0")
// raw - The raw JSON payload bytes to validate.
//
// Returns an error if the payload is invalid against the schema, or nil if valid.
func (ss *SchemaStore) Validate(ctx context.Context, schema string, raw []byte) error {
	schemaPath := strings.TrimLeft(schema, "iglu:")

	rs, ok := ss.cache[schemaPath]

	if !ok {
		rs = new(jsonschema.Schema)

		schemaData, err := schemas.Content.ReadFile(schemaPath)
		if err != nil {
			return fmt.Errorf("failed to read schema file: %w", err)
		}

		err = json.Unmarshal(schemaData, rs)
		if err != nil {
			return fmt.Errorf("failed to marshall schema: %w", err)
		}

		ss.cache[schemaPath] = rs
	}

	validateErrors, err := rs.ValidateBytes(ctx, raw)
	if err != nil {
		return fmt.Errorf("failed to marshall schema: %w", err)
	}

	if len(validateErrors) > 0 {
		return convert(validateErrors)
	}

	return nil
}

func convert(validateErrors []jsonschema.KeyError) error {

	errs := make([]error, len(validateErrors))

	for i := range validateErrors {
		errs[i] = validateErrors[i]
	}

	return errors.Join(errs...)
}
