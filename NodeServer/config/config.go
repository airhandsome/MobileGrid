package config

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"strings"
)

func SetupLogger() {
	logFile := viper.GetString("log.file")
	logLevel := viper.GetString("log.level")

	// 设置日志级别
	var err error
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatalf("Invalid log level: %s", logLevel)
	}
	log.SetLevel(level)
	// 创建日志文件
	LogFile, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// 将日志输出到文件
	log.SetOutput(LogFile)

	// 设置日志格式
	log.SetFormatter(&log.JSONFormatter{})
}
func InitConfig() {
	viper.SetConfigName("default")               // 配置文件名（不带扩展名）
	viper.SetConfigType("yaml")                  // 如果配置文件没有扩展名，则需要指定类型
	viper.AddConfigPath("config")                // 指定查找配置文件的路径
	if err := viper.ReadInConfig(); err != nil { // 查找并读取配置文件
		log.Fatalf("Error reading config file, %s", err)
	}
	LoadConfig()

}

var LogFile *os.File
var NodeId string
var NodeName string
var BackendURL string
var WebsocketURL string

func LoadConfig() {
	nodeID := viper.GetString("node.id")
	if nodeID == "" {
		log.Fatal("NODE_ID environment variable is required")
	}

	nodeName := viper.GetString("node.name")
	if nodeName == "" {
		log.Fatal("NODE_NAME environment variable is required")
	}

	backendURL := viper.GetString("backend_url")
	if backendURL == "" {
		log.Fatal("Backend URL is not configured in the config file")
	}

	WebsocketURL = strings.Replace(backendURL, "http", "ws", -1)
}
