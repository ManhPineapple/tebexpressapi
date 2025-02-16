package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const DefaultLogFile = "/tmp/log"

// InitLogger -- init logger
func InitLoggerRotate() *zap.SugaredLogger {
	// Ensure the directory exists
	if err := os.MkdirAll(DefaultLogFile, 0755); err != nil {
		return nil
	}

	// Create the full file path
	filePath := filepath.Join(DefaultLogFile, "ndapi.log")

	// Create a lumberjack logger for log rotation
	lumberjackLogger := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    10, // Megabytes
		MaxBackups: 3,
		MaxAge:     7,    // Days
		Compress:   true, // Compress old logs
	}

	//Create a zapcore.EncoderConfig
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Create zapcore.Encoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	// Create zapcore.Core for stdout
	consoleCore := zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), zapcore.InfoLevel)

	// Create zapcore.Core for file with lumberjack
	fileCore := zapcore.NewCore(encoder, zapcore.AddSync(lumberjackLogger), zapcore.InfoLevel)

	// Combine the cores
	combinedCore := zapcore.NewTee(consoleCore, fileCore)

	// Create a logger with the combined core
	logger := zap.New(combinedCore, zap.AddCaller())

	// Return a sugared logger
	return logger.Sugar()
}

// Close will flush log to file
func CloseRotate(l *zap.SugaredLogger) {
	_ = l.Sync()
}
