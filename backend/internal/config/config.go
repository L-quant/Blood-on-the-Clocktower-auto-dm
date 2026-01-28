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

	// RabbitMQ configuration
	RabbitMQURL string

	// Qdrant (Vector DB) configuration
	QdrantHost       string
	QdrantPort       int
	QdrantCollection string

	// AutoDM configuration
	AutoDMEnabled    bool
	AutoDMLLMBaseURL string
	AutoDMLLMAPIKey  string
	AutoDMLLMModel   string
	AutoDMLLMTimeout time.Duration

	// Game configuration
	DefaultNominationTimeout  time.Duration
	DefaultVoteTimeout        time.Duration
	DefaultDiscussionDuration time.Duration
	DefaultNightActionTimeout time.Duration
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

		// RabbitMQ
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://botc:botc_password@localhost:5672/"),

		// Qdrant Vector DB
		QdrantHost:       getEnv("QDRANT_HOST", "localhost"),
		QdrantPort:       getEnvInt("QDRANT_PORT", 6333),
		QdrantCollection: getEnv("QDRANT_COLLECTION", "botc_rules"),

		// AutoDM: AI Storyteller configuration
		AutoDMEnabled:    getEnvBool("AUTODM_ENABLED", false),
		AutoDMLLMBaseURL: getEnv("AUTODM_LLM_BASE_URL", "https://api.openai.com/v1"),
		AutoDMLLMAPIKey:  getEnv("AUTODM_LLM_API_KEY", ""),
		AutoDMLLMModel:   getEnv("AUTODM_LLM_MODEL", "gpt-4o"),
		AutoDMLLMTimeout: time.Duration(getEnvInt("AUTODM_LLM_TIMEOUT_SEC", 60)) * time.Second,

		// Game timing configuration
		DefaultNominationTimeout:  time.Duration(getEnvInt("NOMINATION_TIMEOUT_SEC", 10)) * time.Second,
		DefaultVoteTimeout:        time.Duration(getEnvInt("VOTE_TIMEOUT_SEC", 3)) * time.Second,
		DefaultDiscussionDuration: time.Duration(getEnvInt("DISCUSSION_DURATION_SEC", 180)) * time.Second,
		DefaultNightActionTimeout: time.Duration(getEnvInt("NIGHT_ACTION_TIMEOUT_SEC", 30)) * time.Second,
	}
}
