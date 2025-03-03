package main

import (
	"log"
	"shows/internal/bot"
	"shows/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	b, err := bot.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err = b.Start(); err != nil {
		log.Fatal(err)
	}
}
