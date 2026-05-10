package config

import "os"

type Config struct {
	ListenAddr string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBDriver   string
}

func Get() *Config {
	listenAddr := os.Getenv("HTTP_LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":4000"
	}
	return &Config{
		ListenAddr: listenAddr,
		DBDriver:   os.Getenv("DB_DRIVER"),
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
	}
}
