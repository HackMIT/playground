package config

import (
	"log"
	"github.com/spf13/viper"
)

var config *viper.Viper

func Init(env string) {
	var err error

	// Load config from config/base.json
	config = viper.New()
	config.SetConfigName("base")
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
