package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	_ "github.com/lib/pq"
)

var (
	dynamodbClient *dynamodb.Client
	db             *sql.DB
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Unable to load SDK config, %v", err)
	}

	dynamodbClient = dynamodb.NewFromConfig(cfg)

	// Initialize PostgreSQL connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Println("DATABASE_URL environment variable not set, skipping PostgreSQL initialization")
		return
	}

	// Connect to PostgreSQL
	var dbErr error
	db, dbErr = sql.Open("postgres", dbURL)
	if dbErr != nil {
		log.Printf("Failed to connect to PostgreSQL database: %v", dbErr)
		return
	}

	// Test connection
	if pingErr := db.Ping(); pingErr != nil {
		log.Printf("Failed to ping PostgreSQL database: %v", pingErr)
		db = nil
		return
	}

	log.Println("Successfully connected to PostgreSQL database")
}

// Function to add user to PostgreSQL database
func addUserToPostgres(ctx context.Context, userAttributes map[string]string) error {
	if db == nil {
		log.Println("PostgreSQL database connection is not initialized, skipping user addition")
		return nil
	}

	cognitoID := userAttributes["sub"]
	email := userAttributes["email"]
	phone := userAttributes["phone_number"]

	query := `INSERT INTO z_users (cognito_id, email, phone, role) VALUES ($1, $2, $3, $4) ON CONFLICT (cognito_id) DO NOTHING RETURNING user_id`

	role := "user"
	if userRole, exists := userAttributes["custom:role"]; exists && userRole == "admin" {
		role = "admin"
	}

	var userID string
	err := db.QueryRowContext(ctx, query, cognitoID, email, phone, role).Scan(&userID)
	if err != nil {
		log.Printf("Failed to add user to PostgreSQL database: %v", err)
		return err
	}

	log.Printf("User added to PostgreSQL database with ID: %s", userID)
	return nil
}

func handler(ctx context.Context, event events.CognitoEventUserPoolsPostConfirmation) (events.CognitoEventUserPoolsPostConfirmation, error) {
	log.Println(event)

	userAttributes := event.Request.UserAttributes
	userID := userAttributes["sub"]
	userStatus := userAttributes["cognito:user_status"]
	email := userAttributes["email"]
	phoneNumber := userAttributes["phone_number"]

	log.Println(userID)

	if err := addUserToPostgres(ctx, userAttributes); err != nil {
		log.Printf("Failed to add user to PostgreSQL: %v", err)
	}

	item := dynamodb.PutItemInput{
		TableName: aws.String("user_table"),
		Item: map[string]types.AttributeValue{
			"userId":      &types.AttributeValueMemberS{Value: userID},
			"email":       &types.AttributeValueMemberS{Value: email},
			"phoneNumber": &types.AttributeValueMemberS{Value: phoneNumber},
			"userStatus":  &types.AttributeValueMemberS{Value: userStatus},
		},
		ConditionExpression: aws.String("attribute_not_exists(#email)"),
		ExpressionAttributeNames: map[string]string{
			"#email": "email",
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
