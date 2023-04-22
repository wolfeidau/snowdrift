package collector

import (
	"context"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/aws/aws-sdk-go-v2/service/firehose/types"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/snowdrift/internal/flags"
	"github.com/wolfeidau/snowdrift/internal/registry"
)

const (
	collectorPayloadSchema = "iglu:au.id.wolfe.snowplow/CollectorPayload/jsonschema/1-0-0"
)

type Params struct {
	Flags       flags.Service
	FirehoseSvc *firehose.Client
	SchemaStore *registry.SchemaStore
}

func GetHandler(p Params) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusNotImplemented, "nope")
	}
}

// PostHandler handles incoming Snowplow collector payloads and sends them to
// Amazon Kinesis Firehose.
//
// p - Handler parameters
//
// Returns an HTTP 200 response with an "ok" message on success, or an error if
// the payload is invalid or failed to send.
func PostHandler(p Params) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		// limit the analytics payload to limit the impact of abuse or denial of service
		lr := io.LimitReader(c.Request().Body, p.Flags.PayloadLimit)

		data, err := io.ReadAll(lr)
		if err != nil {
			return err
		}

		valid, err := loadAndValidatePayload(ctx, p.SchemaStore, data)
		if err != nil {
			return err
		}

		if valid {
			res, err := p.FirehoseSvc.PutRecord(ctx, &firehose.PutRecordInput{
				DeliveryStreamName: aws.String(p.Flags.FirehoseStreamName),
				Record:             &types.Record{Data: data},
			})
			if err != nil {
				return err
			}

			log.Ctx(ctx).Debug().Str("msg", *res.RecordId).Msg("record sent")
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "ok"})
	}
}

func RedirectHandler(p Params) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusNotImplemented, "nope")
	}
}

func loadAndValidatePayload(ctx context.Context, ss *registry.SchemaStore, raw []byte) (bool, error) {
	// validate the initial collector payload
	err := ss.Validate(ctx, collectorPayloadSchema, raw)
	if err != nil {
		return false, err
	}

	// parse the payload
	scp, err := BuildSnowplowPayload[SnowplowCollectorPayload](ctx, raw)
	if err != nil {
		return false, err
	}

	// validate the inner payload
	err = ss.Validate(ctx, scp.Schema, scp.Data)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("schema", scp.Schema).Msg("failed to validate data")

		// any sort of validation error is ignored as what would the client do about it?
		return false, nil
	}

	return true, nil
}
