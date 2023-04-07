data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
  backend "s3" {
    encrypt = true
  }
}

provider "aws" {
  region = "us-west-2"

  skip_metadata_api_check     = true
  skip_region_validation      = true
  skip_credentials_validation = true
  skip_requesting_account_id  = true

  default_tags {
    tags = {
      Application = "${var.app_name}"
      Stage       = "${var.stage}"
      Branch      = "${var.branch}"
      ManagedBy   = "terraform"
    }
  }
}

resource "random_id" "unique_suffix" {
  byte_length = 2
}

locals {
  app_id = "${lower(var.app_name)}-${lower(var.stage)}-${lower(var.branch)}-${random_id.unique_suffix.hex}"
}

#
# Firehose
#

resource "aws_s3_bucket" "events" {
  bucket = "${local.app_id}-bucket"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "events" {
  bucket = aws_s3_bucket.events.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "events" {
  bucket = aws_s3_bucket.events.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_acl" "events" {
  bucket = aws_s3_bucket.events.id
  acl    = "private"
}

data "aws_iam_policy_document" "firehose_assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["firehose.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

data "aws_iam_policy_document" "firehose_bucket" {
  statement {
    effect = "Allow"

    actions = [
      "s3:AbortMultipartUpload",
      "s3:GetBucketLocation",
      "s3:GetObject",
      "s3:ListBucket",
      "s3:ListBucketMultipartUploads",
      "s3:PutObject",
    ]

    resources = [
      aws_s3_bucket.events.arn,
      "${aws_s3_bucket.events.arn}/*",
    ]
  }
}

resource "aws_iam_role" "firehose" {
  assume_role_policy = data.aws_iam_policy_document.firehose_assume_role.json

  inline_policy {
    name = "s3"

    policy = data.aws_iam_policy_document.firehose_bucket.json
  }

}

resource "aws_kinesis_firehose_delivery_stream" "snowplow" {
  name        = "${local.app_id}-stream"
  destination = "extended_s3"

  extended_s3_configuration {
    role_arn            = aws_iam_role.firehose.arn
    bucket_arn          = aws_s3_bucket.events.arn
    buffer_size         = 128
    buffer_interval     = 900
    compression_format  = "GZIP"
    prefix              = "snowplow/region=${data.aws_region.current.name}/year=!{timestamp:YYYY}/month=!{timestamp:MM}/day=!{timestamp:dd}/hour=!{timestamp:HH}/"
    error_output_prefix = "Errors/us-west-2/!{firehose:random-string}/!{firehose:error-output-type}/!{timestamp:yyyy/MM/dd}/"
  }
}

#
# Lambda
#

data "archive_file" "lambda_zip" {
  type        = "zip"
  source_file = "../build/snowplow-collector/bootstrap"
  output_path = "../build/snowplow-collector/bootstrap.zip"
}

resource "aws_cloudwatch_log_group" "lambda_log_group" {
  name              = "/aws/lambda/${local.app_id}-lambda"
  retention_in_days = 7
}

data "aws_iam_policy_document" "assume_lambda_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "allow_lambda_logging" {
  statement {
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents",
      "logs:DescribeLogStreams",
    ]

    resources = [
      "${aws_cloudwatch_log_group.lambda_log_group.arn}:*",
    ]
  }
}

data "aws_iam_policy_document" "allow_lambda_firehose" {
  statement {
    actions = [
      "firehose:PutRecord",
      "firehose:PutRecordBatch",
    ]
    effect = "Allow"
    resources = [
      aws_kinesis_firehose_delivery_stream.snowplow.arn,
    ]
  }
}

resource "aws_iam_role" "lambda_role" {
  name               = "${local.app_id}-role"
  assume_role_policy = data.aws_iam_policy_document.assume_lambda_role.json

  inline_policy {
    name = "logs"

    policy = data.aws_iam_policy_document.allow_lambda_logging.json
  }

  inline_policy {
    name = "firehose"

    policy = data.aws_iam_policy_document.allow_lambda_firehose.json
  }
}

resource "aws_lambda_function" "snowdrift" {
  filename         = data.archive_file.lambda_zip.output_path
  function_name    = "${local.app_id}-lambda"
  handler          = "bootstrap"
  source_code_hash = base64sha256(data.archive_file.lambda_zip.output_path)
  runtime          = "provided.al2"
  role             = aws_iam_role.lambda_role.arn
  architectures    = ["arm64"]

  environment {
    variables = {
      HOSTING              = "serverless"
      FIREHOSE_STREAM_NAME = "${local.app_id}-stream"
    }
  }
}

#
# APIGW
#

resource "aws_apigatewayv2_api" "lambda" {
  name          = "${local.app_id}-gw"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_stage" "snowdrift" {
  api_id      = aws_apigatewayv2_api.lambda.id
  name        = "$default"
  auto_deploy = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.api_gw.arn

    format = jsonencode({
      requestId               = "$context.requestId"
      sourceIp                = "$context.identity.sourceIp"
      requestTime             = "$context.requestTime"
      protocol                = "$context.protocol"
      httpMethod              = "$context.httpMethod"
      resourcePath            = "$context.resourcePath"
      routeKey                = "$context.routeKey"
      status                  = "$context.status"
      responseLength          = "$context.responseLength"
      integrationErrorMessage = "$context.integrationErrorMessage"
      }
    )
  }

  default_route_settings {
    throttling_burst_limit = 10
    throttling_rate_limit  = 10
  }
}

resource "aws_apigatewayv2_integration" "snowdrift" {
  api_id = aws_apigatewayv2_api.lambda.id

  integration_uri        = aws_lambda_function.snowdrift.invoke_arn
  integration_type       = "AWS_PROXY"
  integration_method     = "POST"
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "snowdrift_snowplow_post" {
  api_id = aws_apigatewayv2_api.lambda.id

  route_key = "POST /com.snowplowanalytics.snowplow/tp2"
  target    = "integrations/${aws_apigatewayv2_integration.snowdrift.id}"
}

resource "aws_cloudwatch_log_group" "api_gw" {
  name = "/aws/api_gw/${local.app_id}"

  retention_in_days = 7
}

resource "aws_lambda_permission" "api_gw" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.snowdrift.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "arn:aws:execute-api:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:${aws_apigatewayv2_api.lambda.id}/*/*"
}
