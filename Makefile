build:
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap main.go

update:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap main.go
	zip myFunction.zip bootstrap
	aws lambda update-function-code --function-name cognitoLogUserFn --zip-file fileb://myFunction.zip

update-config:
	aws lambda update-function-configuration --function-name cognitoLogUserFn --environment file://env.json

logs:
	aws logs tail /aws/lambda/cognitoLogUserFn --follow
