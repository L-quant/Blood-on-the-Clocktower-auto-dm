package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr          string
	WSReadBufferSize  int
	WSWriteBufferSize int
	DBDSN             string
	RedisAddr         string
	JWTSecret         string
	SnapshotInterval  int64
	PrometheusAddr    string
	TraceStdout       bool

	// AutoDM configuration
	AutoDMEnabled    bool
	AutoDMLLMBaseURL string
	AutoDMLLMAPIKey  string
	AutoDMLLMModel   string
	AutoDMLLMTimeout time.Duration
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func Load() Config {
	return Config{
		HTTPAddr:          getEnv("HTTP_ADDR", ":8080"),
		WSReadBufferSize:  getEnvInt("WS_READ_BUFFER", 4096),
		WSWriteBufferSize: getEnvInt("WS_WRITE_BUFFER", 4096),
		DBDSN:             getEnv("DB_DSN", "root:password@tcp(localhost:3316)/agentdm?parseTime=true&multiStatements=true&charset=utf8mb4&collation=utf8mb4_unicode_ci"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6389"),
		JWTSecret:         getEnv("JWT_SECRET", "dev-secret-change"),
		SnapshotInterval:  int64(getEnvInt("SNAPSHOT_INTERVAL", 50)),
		PrometheusAddr:    getEnv("PROM_ADDR", ":9090"),
		TraceStdout:       getEnvBool("TRACE_STDOUT", true),

		// AutoDM: AI Storyteller configuration
		AutoDMEnabled:    getEnvBool("AUTODM_ENABLED", false),
		AutoDMLLMBaseURL: getEnv("AUTODM_LLM_BASE_URL", "https://api.openai.com/v1"),
		AutoDMLLMAPIKey:  getEnv("AUTODM_LLM_API_KEY", ""),
		AutoDMLLMModel:   getEnv("AUTODM_LLM_MODEL", "gpt-4o"),
		AutoDMLLMTimeout: time.Duration(getEnvInt("AUTODM_LLM_TIMEOUT_SEC", 60)) * time.Second,
	}
}
