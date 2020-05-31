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

	flag.Usage = func() {
		fmt.Println("Usage: server -e {mode}")
		os.Exit(1)
	}

	flag.Parse()

	config.Init(*environment)
	db.Init()
	server.Init()
}
