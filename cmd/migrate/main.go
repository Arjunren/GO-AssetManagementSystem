package main

import (
	"context"
	"github.com/Arjunren/asset-management-system/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"os"
)

func main() {
	u := os.Getenv("DATABASE_URL")
	if u == "" {
		log.Fatal("DATABASE_URL is required")
	}
	c := context.Background()
	db, e := pgxpool.New(c, u)
	if e != nil {
		log.Fatal(e)
	}
	defer db.Close()
	if e = migrations.Apply(c, db); e != nil {
		log.Fatal(e)
	}
	log.Print("migrations applied")
}
