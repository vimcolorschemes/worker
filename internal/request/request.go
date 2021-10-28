package request

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

// Post sends a POST HTTP request and returns the response body
func Post(url string, data map[string]string) interface{} {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

	response, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal(err)
	}

	var result map[string]interface{}

	json.NewDecoder(response.Body).Decode(&result)

	return result["json"]
}
