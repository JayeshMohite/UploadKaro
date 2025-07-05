package middleware

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

// LoadConfig loads a JSON configuration file and sets each field as an environment variable.
func LoadConfig(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Decode JSON into a generic map to handle any string-based configuration
	var config map[string]interface{}
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return err
	}

	// Set each field as an environment variable, treating all values as strings
	for key, value := range config {
		switch v := value.(type) {
		case string:
			os.Setenv(key, v)
		case []interface{}:
			// Join array elements into a single comma-separated string
			strValues := make([]string, len(v))
			for i, item := range v {
				strValues[i] = item.(string)
			}
			os.Setenv(key, strings.Join(strValues, ","))
		default:
			log.Printf("Unsupported type for key %s, expected string or array of strings", key)
		}
	}

	return nil
}
