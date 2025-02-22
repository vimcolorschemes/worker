package file

import (
	"log"
	"os"
)

// GetLocalFileContent returns the file content of a local file at a path
func GetLocalFileContent(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// AppendToFile adds content to a local file
func AppendToFile(content string, path string) error {
	log.Printf("Appending to %s: %s", path, content)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	return file.Sync()
}
