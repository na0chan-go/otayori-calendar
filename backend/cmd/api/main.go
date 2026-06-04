package main

import (
	"log"

	"github.com/na0chan-go/otayori-calendar/backend/internal/config"
	"github.com/na0chan-go/otayori-calendar/backend/internal/database"
	"github.com/na0chan-go/otayori-calendar/backend/internal/handler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}

	server := handler.NewServer(cfg, db)
	if err := server.Start(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
