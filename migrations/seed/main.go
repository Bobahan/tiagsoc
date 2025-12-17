package main

import (
	"log"
	"social/internal/config"
	"social/internal/storage"
	"social/internal/storage/postgresql"
)

func main() {
	config := config.New()

	db, err := postgresql.New(config)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	store := storage.New(db)

	storage.Seed(store)
}
