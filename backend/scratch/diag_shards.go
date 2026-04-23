package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/bandhannova/api-hunter/internal/security"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	masterKey := "bdn-bandhannova-master-key-ehb66vk7jhbfl4kjzufg7a456734twrddsbh67363vxfdy64gvaghase32rdvuz"
	
	urls := []string{
		"libsql://bfobs-shard-2-bfobs-shard-2.aws-ap-south-1.turso.io",
		"libsql://bfobs-shard-3-bfobs-shard-3.aws-ap-south-1.turso.io",
		"libsql://bfobs-shard-4-bfobs-shard-4.aws-ap-south-1.turso.io",
		"libsql://bfobs-shard-5-bfobs-shard-5.aws-ap-south-1.turso.io",
		"libsql://bfobs-shard-6-bfobs-shard-6.aws-ap-south-1.turso.io",
	}
	tokens := []string{
		"eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJhIjoicnciLCJpYXQiOjE3NzI2MDI1OTMsImlkIjoiMDE5Y2I3NTgtZjUwMS03MzRkLTk5NzMtMzMyZGExYjk3ZWFhIiwicmlkIjoiODQ3MzJhMzMtOGVlMC00M2QxLWJlNTUtYWYxZGU0OWQ2NGJjIn0.txoUXP2BOoNe3N2fPVUEx2WresSgbsvMeB89fQ784f9Vxf0vdD4nKBXIYbFnGViqBaMPnsbFQekU3JeSaT_pDQ",
		"eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJhIjoicnciLCJpYXQiOjE3NzI2MDMwMDgsImlkIjoiMDE5Y2I3NWYtNDYwMS03ODdmLThiMzItOGE0ZmNkMDMzNDc0IiwicmlkIjoiNTE3MWQzZjYtNmEyMi00NzAwLTk0ODQtYzcxYzQzNjg0ZGZhIn0.2V_cKZCX-LwK69mIo6UUcnX-RqwURrQbsZkO1OHWlXkfON2f1K21RQTc1vbzdf3r3otihh5vQsCjQDc53ZggBg",
		"eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJhIjoicnciLCJpYXQiOjE3NzI2MDMyMzEsImlkIjoiMDE5Y2I3NjItOTcwMS03NDYyLWE4ZWEtNjYzOTA1YmM2NTQxIiwicmlkIjoiOWEyZTRjY2UtNzgxYy00MzBkLTkxN2YtNWExYWQ5MDZmMDFmIn0.UGN4ggWwGnvBcwTP9QhesHcKIS5D0m7lOr49IETS8LcMEShqOm1BtTmuQB1fXLsnINz_sAIfV_YQiCxAX9W6Aw",
		"eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJhIjoicnciLCJpYXQiOjE3NzI2MDM4NjYsImlkIjoiMDE5Y2I3NmMtNmMwMS03NzFjLWJjNzItY2NjMDBmNDZjMzdkIiwicmlkIjoiMDZiYTUwZjMtNTU4MC00YzI2LTljMGUtYjZlMTdjYzA4YzFmIn0.3OnreLm6zzFAAOvglvdinUVcRZalWJ7CrEVySxDlqbjpfMJ0l2JhfGrN8xk90MHxJuZjRr03KpMWHUQbD7kdDw",
		"eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJhIjoicnciLCJpYXQiOjE3NzI2MDQzMTYsImlkIjoiMDE5Y2I3NzMtNDIwMS03MWIxLWExNDEtYmU3ZTA4YjNhNDc2IiwicmlkIjoiZWNiYzYxYTUtOTNlZC00ZjZlLThmNzQtZDkwNjM2MTU1ZWZkIn0.a28gGw0rQdn1lpJRguA3hfY5HivBqYQaeh6wi5nT6HdCPzKCBvjT2TjUC1wKxqCJN1BbCEPwLizHldNYbKy0CQ",
	}

	for i, url := range urls {
		fmt.Printf("\n🌐 Checking Global Manager Shard %d: %s\n", i+1, url)
		connStr := fmt.Sprintf("%s?authToken=%s", url, tokens[i])
		db, err := sql.Open("libsql", connStr)
		if err != nil {
			fmt.Printf("❌ Failed to open connection: %v\n", err)
			continue
		}

		err = db.PingContext(context.Background())
		if err != nil {
			fmt.Printf("❌ Connection FAILED: %v\n", err)
			db.Close()
			continue
		}

		rows, err := db.Query("SELECT name, db_url, encrypted_token FROM managed_databases")
		if err != nil {
			fmt.Printf("❌ Failed to query: %v\n", err)
			db.Close()
			continue
		}

		found := false
		for rows.Next() {
			found = true
			var name, dURL, encrypted string
			rows.Scan(&name, &dURL, &encrypted)
			fmt.Printf("  📦 Managed DB: %s (%s)\n", name, dURL)

			decrypted, err := security.Decrypt(encrypted, masterKey)
			if err != nil {
				fmt.Printf("    ❌ Decryption FAILED: %v\n", err)
				continue
			}
			fmt.Printf("    ✅ Decryption SUCCESS!\n")

			testConn := fmt.Sprintf("%s?authToken=%s", dURL, decrypted)
			testDB, _ := sql.Open("libsql", testConn)
			err = testDB.PingContext(context.Background())
			if err != nil {
				fmt.Printf("    ❌ Connectivity FAILED: %v\n", err)
			} else {
				fmt.Printf("    🚀 Connectivity SUCCESS!\n")
			}
			testDB.Close()
		}
		if !found {
			fmt.Println("  (No managed databases found on this shard)")
		}
		rows.Close()
		db.Close()
	}
}
