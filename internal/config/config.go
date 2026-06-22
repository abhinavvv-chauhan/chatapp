package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Environment string `mapstructure:"ENVIRONMENT"`
	ServerPort  string `mapstructure:"SERVER_PORT"`
	DBUrl       string `mapstructure:"DATABASE_URL"`
	RedisUrl    string `mapstructure:"REDIS_URL"`
	JWTSecret   string `mapstructure:"JWT_SECRET"` 
}


func LoadConfig() (Config, error) {
	var config Config

	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	viper.BindEnv("DATABASE_URL")
	viper.BindEnv("JWT_SECRET")
	viper.BindEnv("REDIS_URL")
	viper.BindEnv("SERVER_PORT")
	viper.BindEnv("PORT")
	viper.BindEnv("ENVIRONMENT")

	err := viper.ReadInConfig()
	if err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		return config, err
	}

	return config, nil
}