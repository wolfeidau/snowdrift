package main

import (
	"context"
	"io"
	"net/http"

	"github.com/alecthomas/kong"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	middleware "github.com/wolfeidau/echo-middleware"
	"github.com/wolfeidau/lambda-go-extras/lambdaextras"
	lmw "github.com/wolfeidau/lambda-go-extras/middleware"
	"github.com/wolfeidau/lambda-go-extras/standard"
	"github.com/wolfeidau/snowdrift/internal/collector"
	"github.com/wolfeidau/snowdrift/internal/flags"
	"github.com/wolfeidau/snowdrift/internal/registry"
)

var (
	version = "unknown"

	cli flags.Service
)

func main() {
	kong.Parse(&cli,
		kong.Vars{"version": version}, // bind a var for version
	)

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal().Err(err).Msg("config failed")
	}

	svc := firehose.NewFromConfig(cfg)

	e := echo.New()

	e.HTTPErrorHandler = customHTTPErrorHandler

	p := collector.Params{
		Flags:       cli,
		FirehoseSvc: svc,
		SchemaStore: registry.NewSchemaStore(),
	}

	e.POST("/com.snowplowanalytics.snowplow/tp2", collector.PostHandler(p))

	// TODO: implement these endpoints
	// e.GET("/i", collector.GetHandler(p))
	// e.GET("/r/tp2", collector.RedirectHandler(p))

	flds := lmw.FieldMap{"version": version}

	switch cli.Hosting {
	case "container":
		container(e, flds)
	case "serverless":
		serverless(e, flds)
	default:
		log.Fatal().Str("hosting", cli.Hosting).Msg("invalid hosting option")
	}
}

func container(e *echo.Echo, flds lmw.FieldMap) {
	e.Logger.SetOutput(io.Discard)

	e.Use(middleware.ZeroLogWithConfig(
		middleware.ZeroLogConfig{
			Fields: flds,
			Level:  zerolog.InfoLevel,
		},
	))

	e.Use(middleware.ZeroLogRequestLog())
	e.Use(echomiddleware.Gzip())

	log.Fatal().Str("hosting", cli.Hosting).Msg("starting http listener")

	log.Fatal().Err(e.Start(":3333")).Msg("listener failed")
}

func serverless(e *echo.Echo, flds lmw.FieldMap) {

	h := lambdaextras.GenericHandler(httpadapter.NewV2(e.Server.Handler).ProxyWithContext)

	standard.Default(h, standard.Fields(flds))
}

func customHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	log.Ctx(c.Request().Context()).Error().Err(err).Msg("failed to process request")

	he, ok := err.(*echo.HTTPError)
	if ok {
		if he.Internal != nil {
			if herr, ok := he.Internal.(*echo.HTTPError); ok {
				he = herr
			}
		}
	} else {
		he = &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: http.StatusText(http.StatusInternalServerError),
		}
	}
}
