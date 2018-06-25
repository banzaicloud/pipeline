package main

import (
	"context"
	"os"

	"github.com/banzaicloud/pipeline/client"
)

// First you have to create a Pipeline Bearer token and put it into the TOKEN env variable.
func main() {
	config := client.NewConfiguration()
	ctx := context.WithValue(context.Background(), client.ContextAccessToken, os.Getenv("TOKEN"))
	pipeline := client.NewAPIClient(config)

	tokenRequest := client.TokenCreateRequest{Name: "drone token", VirtualUser: "banzaicloud/pipeline"}
	tokenResponse, _, err := pipeline.AuthApi.GenerateToken(ctx, tokenRequest)

	if err != nil {
		panic(err)
	}

	// Overwrite the existing context token
	ctx = context.WithValue(context.Background(), client.ContextAccessToken, tokenResponse.Token)

	// Create a new Generic secret
	secretRequest := client.CreateSecretRequest{
		Name:   "my-password",
		Type:   "generic",
		Values: map[string]interface{}{"password": "s3cr3t"},
		Tags:   []string{"banzai:hidden"},
	}

	secretResponse, _, err := pipeline.SecretsApi.AddSecrets(ctx, 2, secretRequest)
	if err != nil {
		panic(err)
	}

	println("Secret id:", secretResponse.Id)
}
