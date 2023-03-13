output "function_name" {
  description = "Name of the Lambda function."

  value = aws_lambda_function.snowdrift.function_name
}

output "api_url" {
  value = aws_apigatewayv2_stage.snowdrift.invoke_url
}