package main

import (
	"log"

	"lucky-seven/backend/internal/app"
	"lucky-seven/backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	app.New(cfg).Run()
}
