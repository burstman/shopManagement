package config

import "os"

type Config struct {
	ListenAddr    string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBDriver      string
	DatabaseURL   string
}

func Get() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}

	cfg := &Config{
		ListenAddr:    ":" + port,
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		DBDriver:      os.Getenv("DB_DRIVER"),
		DBHost:        os.Getenv("DB_HOST"),
		DBPort:        os.Getenv("DB_PORT"),
		DBUser:        os.Getenv("DB_USER"),
		DBPassword:    os.Getenv("DB_PASSWORD"),
		DBName:        os.Getenv("DB_NAME"),
	}

	return cfg
}
