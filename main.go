package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var dynamodbClient *dynamodb.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Unable to load SDK config, %v", err)
	}

	dynamodbClient = dynamodb.NewFromConfig(cfg)
}

func handler(ctx context.Context, event events.CognitoEventUserPoolsPostConfirmation) (events.CognitoEventUserPoolsPostConfirmation, error) {

	log.Println(event)

	userAttributes := event.Request.UserAttributes
	userID := userAttributes["sub"]
	userStatus := userAttributes["cognito:user_status"]
	email := userAttributes["email"]
	phoneNumber := userAttributes["phone_number"]

	log.Println(userID)

	item := dynamodb.PutItemInput{
		TableName: aws.String("user_table"),
		Item: map[string]types.AttributeValue{
			"userId":      &types.AttributeValueMemberS{Value: userID},
			"email":       &types.AttributeValueMemberS{Value: email},
			"phoneNumber": &types.AttributeValueMemberS{Value: phoneNumber},
			"userStatus":  &types.AttributeValueMemberS{Value: userStatus},
		},
	}

	_, err := dynamodbClient.PutItem(ctx, &item)
	if err != nil {
		log.Fatalf("Unable to add item to dynamodDb User table, %v", err)
	}

	id := generateTimeID()

	entityItem := dynamodb.PutItemInput{
		TableName: aws.String("entityTable"),
		Item: map[string]types.AttributeValue{
			"id":     &types.AttributeValueMemberS{Value: id},
			"name":   &types.AttributeValueMemberS{Value: userAttributes["name"]},
			"idType": &types.AttributeValueMemberS{Value: userID},
			"type":   &types.AttributeValueMemberS{Value: "user"},
		},
	}

	_, err = dynamodbClient.PutItem(ctx, &entityItem)
	if err != nil {
		log.Fatalf("Unable to add item to dynamodDb Entity table, %v", err)
	}

	return event, nil
}

func generateTimeID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func main() {
	lambda.Start(handler)
}
