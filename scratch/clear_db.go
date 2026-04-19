package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load("../backend/.env")

	url := os.Getenv("TURSO_GLOBAL_URL")
	token := os.Getenv("TURSO_GLOBAL_TOKEN")

	if url == "" || token == "" {
		log.Fatal("Global DB credentials missing in .env")
	}

	db, err := sql.Open("libsql", fmt.Sprintf("%s?authToken=%s", url, token))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM managed_databases")
	if err != nil {
		log.Fatal("Failed to clear managed_databases:", err)
	}

	fmt.Println("✅ All managed databases removed. Global Manager is now clean!")
}
