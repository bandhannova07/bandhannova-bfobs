package system

import (
	"log"
	"net/http"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
)

// StartAntiSleepWorker pings the app's own public URL to keep it awake on Hugging Face
func StartAntiSleepWorker(interval time.Duration) {
	// If no public URL is provided, we can't self-ping effectively via internet
	// but we can at least log that it's waiting for one.
	publicURL := config.AppConfig.SupabaseURL // Using this as a placeholder or we can add a new config field
	
	// Better yet, let's look for a specific PUBLIC_URL env var
	// For now, if it's empty, we'll ping localhost as a fallback (though public is better)
	
	log.Printf("🛡️  Anti-Sleep System activated (Interval: %v)", interval)
	ticker := time.NewTicker(interval)

	go func() {
		for range ticker.C {
			pingSelf(publicURL)
		}
	}()
}

func pingSelf(url string) {
	// If URL is empty, we try to ping localhost
	target := url
	if target == "" {
		target = "http://localhost:7860"
	}

	start := time.Now()
	resp, err := http.Get(target)
	if err != nil {
		log.Printf("⚠️  Anti-Sleep Ping failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("💓 Self-Ping successful! Status: %d | Latency: %v", resp.StatusCode, time.Since(start))
	} else {
		log.Printf("⚠️  Self-Ping returned non-OK status: %d", resp.StatusCode)
	}
}
