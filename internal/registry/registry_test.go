package registry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPageView(t *testing.T) {

	assert := require.New(t)

	jsonData := []byte(`[{"e": "pp","tv": "js-3.8.0","p": "web"}]`)

	ss := NewSchemaStore()

	err := ss.Validate(context.TODO(), "com.snowplowanalytics.snowplow/payload_data/jsonschema/1-0-4", jsonData)
	assert.NoError(err)
}
