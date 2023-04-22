package flags

import "github.com/alecthomas/kong"

type Service struct {
	Version            kong.VersionFlag
	RawEventLogging    bool   `help:"Enable raw event logging." env:"RAW_EVENT_LOGGING"`
	Debug              bool   `help:"Enable debug logging." env:"DEBUG"`
	Hosting            string `help:"The environment the service is hosted within, which is either serverless or container." env:"HOSTING"`
	FirehoseStreamName string `help:"The name of the firehose delivery stream which receives valid events." env:"FIREHOSE_STREAM_NAME"`
	PayloadLimit       int64  `help:"The payload limit for analytics events" env:"PAYLOAD_LIMIT" default:"10000"`
}
