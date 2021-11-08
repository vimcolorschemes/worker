package request

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

// Post sends a POST HTTP request and returns the response body
func Post(url string, data map[string]string) (interface{}, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

	response, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal(err)
	}

	var result map[string]interface{}

	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result["json"], nil
}
