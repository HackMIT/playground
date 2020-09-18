package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Secret string

const (
	TwilioAccountSID Secret = "TWILIO_ACCOUNT_SID"
	TwilioAuthToken         = "TWILIO_AUTH_TOKEN"
	YouTubeKey              = "YOUTUBE_API_KEY"
)

var config *viper.Viper

func Init(env string) {
	var err error

	// Load environment variables
	godotenv.Load()

	// Load config from config/base.json
	config = viper.New()
	config.SetConfigName("base")
	config.AddConfigPath(".")
	config.AddConfigPath("config")
	err = config.ReadInConfig()

	if err != nil {
		log.Println(err)
		log.Fatal("Unable to process base config file. Make sure the " +
			"file contains valid JSON and is in the correct " +
			"location.")
	}

	// Allow for environment-specific configs (e.g. dev.json or prod.json)
	config.SetConfigName(env)
	config.MergeInConfig()
}

func GetConfig() *viper.Viper {
	return config
}

func GetSecret(key Secret) string {
	return os.Getenv(string(key))
}
