package database

import (
	"encoding/json"
	"log"
	"time"
)

// CreateReport stores a job report in the database.
func CreateReport(job string, elapsedTime float64, data map[string]interface{}) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("INSERT INTO reports (date, job, elapsed_time, data) VALUES (?, ?, ?, ?)",
		time.Now(), job, elapsedTime, string(dataJSON))
	if err != nil {
		log.Printf("Error creating report: %s", err)
		panic(err)
	}
}
