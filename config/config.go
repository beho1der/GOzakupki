package config

import (
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
	"time"
)

type TimersConfig struct {
	Update time.Duration
}

type MainServerConfig struct {
	Proxy    bool
	ProxyUrl string
}

type Config struct {
	Timers        TimersConfig
	LogLevel      logrus.Level
	Main          MainServerConfig
	Port          string
	KeepaliveTime float64
	TimeWait      float64
	AppName       string
	HttpTimeWait  time.Duration
}

// New returns a new Config struct
func New() *Config {
	return &Config{
		AppName: getEnv("APP_NAME", "gozakupki"),
		Timers: TimersConfig{
			Update: getEnvAsTime("UPDATE", time.Duration(20)*time.Second),
		},
		Main: MainServerConfig{
			Proxy:    getEnvAsBool("PROXY_ENABLE", false),
			ProxyUrl: getEnv("PROXY_URL", ""),
		},
		LogLevel:      getEnvAsLogger("LOG_LEVEL", "info"),
		Port:          getEnv("PORT", "8025"),
		KeepaliveTime: getEnvAsFloat64("KEEPALIVETIME", 60),
		TimeWait:      getEnvAsFloat64("TIMEWAIT", 20),
		HttpTimeWait:  getEnvAsTime("HTTP_TIME_WAIT", time.Duration(5)*time.Second),
	}
}

// Simple helper function to read an environment or return a default value
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

// Simple helper function to read an environment variable into integer or return a default value
func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

// Simple helper function to read an environment variable into integer64 or return a default value
func getEnvAsInt64(name string, defaultVal int64) int64 {
	valueStr := getEnv(name, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultVal
}

func getEnvNameSpaces(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return "/" + value + "/"
	}
	return "/" + defaultVal + "/"
}

// Helper to read an environment variable into a bool or return default value
func getEnvAsBool(name string, defaultVal bool) bool {
	valStr := getEnv(name, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}

// Helper to read an environment variable into a float64 or return default value
func getEnvAsFloat64(name string, defaultVal float64) float64 {
	valStr := getEnv(name, "")
	if val, err := strconv.ParseFloat(valStr, 64); err == nil {
		return val
	}
	return defaultVal
}

// Helper to read an environment variable into a string slice or return default value
func getEnvAsSlice(name string, defaultVal []string, sep string) []string {
	valStr := getEnv(name, "")
	if valStr == "" {
		return defaultVal
	}
	val := strings.Split(valStr, sep)
	return val
}

// Преобразование в справочник
func getEnvAsMap(name string, defaultVal map[string]string, sep string) map[string]string {
	var m = make(map[string]string)
	valStr := getEnv(name, "")
	if valStr == "" {
		return defaultVal
	}
	val := strings.Split(valStr, sep)
	if len(val) > 1 {
		for i := 0; i < len(val); i++ {
			elem := strings.Split(val[i], ":")
			if len(elem) > 1 {
				m[elem[0]] = elem[1]
			}
		}
	}
	return m
}

func getEnvAsTime(name string, defaultVal time.Duration) time.Duration {
	valStr := getEnv(name, "")
	if value, err := strconv.Atoi(valStr); err == nil {
		time := time.Duration(value) * time.Second
		return time
	}
	return defaultVal
}

func getEnvAsLogger(key string, defaultVal string) logrus.Level {
	var level logrus.Level
	if value, exists := os.LookupEnv(key); exists {
		if level, err := logrus.ParseLevel(value); err == nil {
			return level
		}
	}
	level, _ = logrus.ParseLevel(defaultVal)
	return level
}
