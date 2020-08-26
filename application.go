package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/techx/playground/config"
	"github.com/techx/playground/db"
	"github.com/techx/playground/server"
)

func main() {
	environment := flag.String("e", "dev", "")
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	reset := flag.Bool("reset", false, "Resets the database")

	flag.Usage = func() {
		fmt.Println("Usage: server -e {mode} -p {port}")
		os.Exit(1)
	}

	flag.Parse()

	config.Init(*environment)
	db.Init(*reset)
	server.Init(port)
}
