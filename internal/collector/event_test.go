package collector

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wolfeidau/snowdrift/internal/registry"
)

func TestPageView(t *testing.T) {

	assert := require.New(t)

	jsonData, err := os.ReadFile("data/event_pv.json")
	assert.NoError(err)

	scp, err := BuildSnowplowPayload[SnowplowCollectorPayload](context.TODO(), jsonData)
	assert.NoError(err)
	assert.Equal("iglu:com.snowplowanalytics.snowplow/payload_data/jsonschema/1-0-4", scp.Schema)

	ss := registry.NewSchemaStore()

	err = ss.Validate(context.TODO(), scp.Schema, scp.Data)
	assert.NoError(err)
}
