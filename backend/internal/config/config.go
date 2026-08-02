package config

import "os"

type Config struct {
	Port               string
	DatabaseURL        string
	TigerBeetleAddress string
	TigerBeetleCluster string
	JWTSecret          string
	OpenRouterAPIKey   string
	OpenRouterModel    string
}

func Load() Config {
	return Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		TigerBeetleAddress: getEnv("TIGERBEETLE_ADDRESS", "127.0.0.1:3000"),
		TigerBeetleCluster: getEnv("TIGERBEETLE_CLUSTER_ID", "0"),
		JWTSecret:          getEnv("JWT_SECRET", "dev_secret_change_me"),
		OpenRouterAPIKey:   getEnv("OPENROUTER_API_KEY", ""),
		OpenRouterModel:    getEnv("OPENROUTER_MODEL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
