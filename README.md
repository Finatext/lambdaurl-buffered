# lambdaurl-buffered
`lambdaurl` converts an `http.Handler` into a Lambda request handler. It supports Lambda Function URLs configured with buffered response mode.

Lambda functions with

- API Gateway v1 (REST), API Gateway v2 (HTTP), Application Load Balancer: Use https://github.com/awslabs/aws-lambda-go-api-proxy
- Function URLs (streaming): Use https://github.com/aws/aws-lambda-go/blob/main/lambdaurl/http_handler.go
- Function URLs (buffered): Use this library until the above options add support for buffered responses.

```go
e := echo.New()
e.GET("/hc", HealthCheck)

// lambda from github.com/aws/aws-lambda-go/lambda
lambda.Start(lambdaurl.Wrap(e))
```

## License
This project is a redistribution with modifications of code from: https://github.com/aws/aws-lambda-go/blob/main/lambdaurl/http_handler.go
