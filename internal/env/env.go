package env

import "os"

func init() {
	if os.Getenv("BAML_LOG") == "" {
		os.Setenv("BAML_LOG", "off")
	}
}
