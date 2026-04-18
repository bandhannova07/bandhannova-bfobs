package main

import (
	"fmt"
	"log"
	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
)

func main() {
	config.LoadConfig()
	err := database.InitShardRouter(
		config.AppConfig.TursoAuthURL, config.AppConfig.TursoAuthToken,
		config.AppConfig.TursoAnalyticsURL, config.AppConfig.TursoAnalyticsToken,
		config.AppConfig.TursoGlobalURL, config.AppConfig.TursoGlobalToken,
		config.AppConfig.TursoUserShardURLs, config.AppConfig.TursoUserShardTokens,
	)
	if err != nil {
		log.Fatal(err)
	}

	rows, err := database.Router.GetGlobalManagerDB().Query(`
		SELECT s.name, c.name, c.id, (SELECT COUNT(*) FROM managed_api_keys WHERE card_id = c.id AND status = 'active' AND is_deleted = 0)
		FROM api_cards c 
		JOIN api_sections s ON c.section_id = s.id
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("API CARDS & KEY COUNTS:")
	for rows.Next() {
		var sname, cname, cid string
		var count int
		rows.Scan(&sname, &cname, &cid, &count)
		
		fmt.Printf("[%s] %s (ID: %s) -> Active Keys: %d\n", sname, cname, cid, count)
	}
}
