resource "aws_api_gateway_rest_api" "samsverynice" {
  name        = "samsverynice"
  description = "its a nice website"
  endpoint_configuration {
    types = [
      "REGIONAL",
    ]
  }
}

resource "aws_api_gateway_resource" "base" {
  rest_api_id = aws_api_gateway_rest_api.samsverynice.id
  parent_id   = ""
  path_part   = ""
}

resource "aws_api_gateway_method" "base_get" {
  rest_api_id   = aws_api_gateway_rest_api.samsverynice.id
  resource_id   = aws_api_gateway_resource.base.id
  http_method   = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "very_nice_lambda" {
  rest_api_id             = aws_api_gateway_rest_api.samsverynice.id
  resource_id             = aws_api_gateway_resource.base.id
  http_method             = aws_api_gateway_method.base_get.http_method
  type                    = "AWS_PROXY"
  integration_http_method = "POST"
  uri                     = aws_lambda_function.very_nice.invoke_arn
}

resource "aws_api_gateway_method_response" "base_get_200" {
  rest_api_id = aws_api_gateway_rest_api.samsverynice.id
  resource_id = aws_api_gateway_resource.base.id
  http_method = aws_api_gateway_method.base_get.http_method
  status_code = "200"
  response_models = {
    "text/plain" = "Empty"
  }
}

resource "aws_api_gateway_deployment" "prod" {
  depends_on  = [aws_api_gateway_integration.very_nice_lambda]
  rest_api_id = aws_api_gateway_rest_api.samsverynice.id
  stage_name  = "prod"
}