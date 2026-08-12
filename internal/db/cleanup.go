package db

import (
	"log"
	"time"
)

func StartCleanupJob() {
	go func() {
		// Run once initially
		runCleanup()

		// Run every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			runCleanup()
		}
	}()
}

func runCleanup() {
	log.Println("Running background cleanup job to remove old heartbeats...")
	
	// Delete heartbeats older than 30 days
	// SQLite syntax for date subtraction
	res, err := DB.Exec(`DELETE FROM heartbeats WHERE checked_at <= datetime('now', '-30 days')`)
	if err != nil {
		log.Println("Error during cleanup:", err)
		return
	}

	rows, _ := res.RowsAffected()
	log.Printf("Cleanup completed. Deleted %d old heartbeats.\n", rows)
}
