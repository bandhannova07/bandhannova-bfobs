package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	dbUrl := "libsql://bfobs-shard-3-bfobs-shard-3.aws-ap-south-1.turso.io?authToken=eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJhIjoicnciLCJpYXQiOjE3NzI2MDMwMDgsImlkIjoiMDE5Y2I3NWYtNDYwMS03ODdmLThiMzItOGE0ZmNkMDMzNDc0IiwicmlkIjoiNTE3MWQzZjYtNmEyMi00NzAwLTk0ODQtYzcxYzQzNjg0ZGZhIn0.2V_cKZCX-LwK69mIo6UUcnX-RqwURrQbsZkO1OHWlXkfON2f1K21RQTc1vbzdf3r3otihh5vQsCjQDc53ZggBg"
	db, err := sql.Open("libsql", dbUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 1. Create BandhanNova BFOBS Product
	_, err = db.Exec(`
		INSERT OR IGNORE INTO managed_products (id, name, slug, app_type, description, status, created_at)
		VALUES (1, 'BandhanNova BFOBS', 'bfobs', 'system', 'Core Infrastructure & Global Registry', 'active', CURRENT_TIMESTAMP)
	`)
	if err != nil {
		log.Printf("Error creating system product: %v", err)
	}

	// 2. Associate Core Shards with BFOBS Product
	// Find the global shard and move it to product 1
	_, err = db.Exec(`
		UPDATE managed_databases 
		SET product_id = 1 
		WHERE name LIKE '%Global%' OR name LIKE '%Master%'
	`)
	if err != nil {
		log.Printf("Error associating shards: %v", err)
	}

	fmt.Println("System product 'BandhanNova BFOBS' initialized and shards migrated.")
}
