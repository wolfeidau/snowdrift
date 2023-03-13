package schemas

import "embed"

// Content holds our static web server content.
//
//go:embed com.snowplowanalytics.snowplow/* au.id.wolfe.snowplow/*
var Content embed.FS
