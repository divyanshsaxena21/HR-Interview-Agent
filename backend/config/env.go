package config

import (
	"os"
)

type Config struct {
	GroqAPIKey  string
	MongoURI    string
	Port        string
	JWTSecret   string
	SMTPHost    string
	SMTPPort    string
	SMTPUser    string
	SMTPPass    string
}

func LoadConfig() Config {
	return Config{
		GroqAPIKey: os.Getenv("GROQ_API_KEY"),
		MongoURI:   os.Getenv("MONGO_URI"),
		Port:       os.Getenv("PORT"),
		JWTSecret:  os.Getenv("JWT_SECRET"),
		SMTPHost:   os.Getenv("SMTP_HOST"),
		SMTPPort:   os.Getenv("SMTP_PORT"),
		SMTPUser:   os.Getenv("SMTP_USER"),
		SMTPPass:   os.Getenv("SMTP_PASS"),
	}
}
