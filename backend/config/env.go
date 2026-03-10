package config

import (
	"os"
)

type Config struct {
	GroqAPIKey      string
	MurfAPIKey      string
	AssemblyAIKey   string
	MongoURI        string
	Port            string
}

func LoadConfig() Config {
	return Config{
		GroqAPIKey:    os.Getenv("GROQ_API_KEY"),
		MurfAPIKey:    os.Getenv("MURF_API_KEY"),
		AssemblyAIKey: os.Getenv("ASSEMBLYAI_API_KEY"),
		MongoURI:      os.Getenv("MONGO_URI"),
		Port:          os.Getenv("PORT"),
	}
}
