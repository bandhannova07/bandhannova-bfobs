package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	// bdn-blogs-shard-1 credentials
	dbURL := "libsql://bdn-blogs-db-bdn-blogs-db-1.aws-ap-south-1.turso.io"
	dbToken := "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJhIjoicnciLCJpYXQiOjE3NzY3NTUxMDEsImlkIjoiMDE5ZGFlZGItMzgwMS03N2NhLWI3OTYtMjYyZDc5ZjIwYWIyIiwicmlkIjoiMWE1YzdlYmItNGZlMy00NDZlLWFhZmMtYzVmNTRhZmVmMzU0In0.F6Yg-zzAHxTj40Kgdu_3rdb6C4qsmSyHMW7vQ4SvP2XWU7M2NETDA5hgDHnMbjIhiTnZu21sG8XlV4cxszC0BQ"

	connStr := fmt.Sprintf("%s?authToken=%s", dbURL, dbToken)
	db, err := sql.Open("libsql", connStr)
	if err != nil {
		log.Fatalf("Failed to open connection: %v", err)
	}
	defer db.Close()

	fmt.Println("🚀 Writing test data to bdn-blogs-shard-1...")

	// 1. Create table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS system_verification_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_name TEXT NOT NULL,
		status TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// 2. Insert data
	events := []struct {
		Name   string
		Status string
	}{
		{"Local Ecosystem Init", "SUCCESS"},
		{"Backend Pulse Sync", "ONLINE"},
		{"Frontend Studio Fix", "APPLIED"},
		{"End-to-End Test", "IN_PROGRESS"},
	}

	for _, e := range events {
		_, err = db.Exec("INSERT INTO system_verification_logs (event_name, status) VALUES (?, ?)", e.Name, e.Status)
		if err != nil {
			fmt.Printf("❌ Failed to insert %s: %v\n", e.Name, err)
		} else {
			fmt.Printf("✅ Inserted: %s\n", e.Name)
		}
	}

	fmt.Println("✨ Data write completed successfully!")
}
