package main

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/viper"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func main() {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	viper.ReadInConfig()

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(viper.GetString("GEMINI_API_KEY")))
	if err != nil { log.Fatal(err) }
	defer client.Close()
	
	iter := client.ListModels(ctx)
	for {
		m, err := iter.Next()
		if err == iterator.Done { break }
		if err != nil { log.Fatal(err) }
		fmt.Println(m.Name)
	}
}
