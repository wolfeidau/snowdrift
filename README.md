# snowdrift

This is suite of services used with [snowplow analytics](https://github.com/snowplow/snowplow).

# Warning

This is very a much a work in progress, lots of work to do tidying up loose ends and tightening up the API, and most importantly adding tests.

# Deployment

This project uses [terraform](https://www.terraform.io/) with some state stored in an s3 bucket I locate using SSM Parameter store located at `/config/$(STAGE)/$(BRANCH)/terraform_bucket`.

For more information on how this works have a read of [Makefile](./Makefile).

To initialism state
```
make init
```

To build the snowdrift-collector lambda.

```
make build-collector
```

To apply the latest terraform changes.
```
make apply
```

All the deployment code is in [infra](./infra).

# Testing

Page View CURL command for testing.

```
curl -X POST https://whatever.execute-api.us-west-2.amazonaws.com/com.snowplowanalytics.snowplow/tp2 -H 'Content-Type: application/json' -d @internal/collector/data/event_pv.json
```

# Links

* https://docs.snowplow.io/docs/collecting-data/collecting-from-own-applications/snowplow-tracker-protocol/
* https://github.com/jazztong/terraform-lambda-go
* https://developer.hashicorp.com/terraform/tutorials/aws/lambda-api-gateway

# License

This application is released under Apache 2.0 license and is copyright [Mark Wolfe](https://www.wolfe.id.au).