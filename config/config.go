package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServerPort             string
	PostgresDSN            string
	RedisAddr              string
	RedisPass              string
	KafkaBrokers           []string
	KafkaTopicStandard     string
	KafkaTopicExpress      string
	KafkaDLTStandard       string
	KafkaDLTExpress        string
	ConsumerGroupID        string
	ConsumerID             string
	DispatcherMode         string
	LockTTLSeconds         int
	SMSPrice               int64
	SMSExpressPrice        int64
	ExpressSLAMs           int
	DispatchMaxRetries     int
	DispatchRetryBackoffMs int
	ProviderMinLatencyMs   int
	ProviderMaxLatencyMs   int
	ProviderFailureRate    float64
	MaxSMSLength           int
}

func LoadConfig() *Config {
	serverPort := getEnv("SERVER_PORT", "8080")
	postgresDSN := getEnv("POSTGRES_DSN", "postgres://sms:sms@localhost:5433/smsdb?sslmode=disable")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPass := os.Getenv("REDIS_PASS")
	kafkaBrokers := splitCSV(getEnv("KAFKA_BROKERS", "localhost:9092"))
	kafkaTopicStandard := getEnv("KAFKA_TOPIC_STANDARD", "sms-standard")
	kafkaTopicExpress := getEnv("KAFKA_TOPIC_EXPRESS", "sms-express")
	kafkaDLTStandard := getEnv("KAFKA_DLT_STANDARD", "sms-standard-dlt")
	kafkaDLTExpress := getEnv("KAFKA_DLT_EXPRESS", "sms-express-dlt")
	consumerGroupID := getEnv("CONSUMER_GROUP_ID", "sms-dispatcher")
	consumerID := getEnv("CONSUMER_ID", "dispatcher-1")
	dispatcherMode := strings.ToLower(getEnv("DISPATCHER_MODE", "standard"))
	lockTTLSeconds := getEnvInt("LOCK_TTL_SECONDS", 30)
	smsPrice := getEnvInt64("SMS_PRICE", 1)
	smsExpressPrice := getEnvInt64("SMS_EXPRESS_PRICE", 2)
	expressSLAMs := getEnvInt("EXPRESS_SLA_MS", 2000)
	dispatchMaxRetries := getEnvInt("DISPATCH_MAX_RETRIES", 3)
	dispatchRetryBackoffMs := getEnvInt("DISPATCH_RETRY_BACKOFF_MS", 200)
	providerMinLatencyMs := getEnvInt("PROVIDER_MIN_LATENCY_MS", 50)
	providerMaxLatencyMs := getEnvInt("PROVIDER_MAX_LATENCY_MS", 200)
	providerFailureRate := getEnvFloat("PROVIDER_FAILURE_RATE", 0.02)
	maxSMSLength := getEnvInt("MAX_SMS_LENGTH", 160)

	return &Config{
		ServerPort:             serverPort,
		PostgresDSN:            postgresDSN,
		RedisAddr:              redisAddr,
		RedisPass:              redisPass,
		KafkaBrokers:           kafkaBrokers,
		KafkaTopicStandard:     kafkaTopicStandard,
		KafkaTopicExpress:      kafkaTopicExpress,
		KafkaDLTStandard:       kafkaDLTStandard,
		KafkaDLTExpress:        kafkaDLTExpress,
		ConsumerGroupID:        consumerGroupID,
		ConsumerID:             consumerID,
		DispatcherMode:         dispatcherMode,
		LockTTLSeconds:         lockTTLSeconds,
		SMSPrice:               smsPrice,
		SMSExpressPrice:        smsExpressPrice,
		ExpressSLAMs:           expressSLAMs,
		DispatchMaxRetries:     dispatchMaxRetries,
		DispatchRetryBackoffMs: dispatchRetryBackoffMs,
		ProviderMinLatencyMs:   providerMinLatencyMs,
		ProviderMaxLatencyMs:   providerMaxLatencyMs,
		ProviderFailureRate:    providerFailureRate,
		MaxSMSLength:           maxSMSLength,
	}
}

func getEnv(key string, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func getEnvInt64(key string, defaultValue int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func getEnvFloat(key string, defaultValue float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
