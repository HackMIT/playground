package config

import (
	"log"
	"github.com/spf13/viper"
)

var config *viper.Viper

func Init(env string) {
	var err error

	config = viper.New()
	config.SetConfigName("base")
	config.AddConfigPath("config")
	err = config.ReadInConfig()

	if err != nil {
		log.Fatal("error on parsing base config file")
	}

	config.SetConfigName(env)
	config.MergeInConfig()

	if err != nil {
		log.Println("error on parsing env config file")
	}
}

func GetConfig() *viper.Viper {
	return config
}
