package common

import (
	"log"

	"github.com/joho/godotenv"
)

func LoadDotenv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Failed to load .env file, using os environment instead.")
	}
}
