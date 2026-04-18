package main

import (
	"fmt"
	"log"
	"strings"
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
		SELECT s.name, c.name, c.endpoint_url 
		FROM api_cards c 
		JOIN api_sections s ON c.section_id = s.id
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("API CARDS & SLUGS:")
	for rows.Next() {
		var sname, cname, url string
		rows.Scan(&sname, &cname, &url)
		
		// Simulate SQL: REPLACE(LOWER(name), ' ', '-')
		sslug := strings.ReplaceAll(strings.ToLower(sname), " ", "-")
		cslug := strings.ReplaceAll(strings.ToLower(cname), " ", "-")
		
		fmt.Printf("[%s] %s -> slugs: /%s/%s/execute\n", sname, cname, sslug, cslug)
	}
}
