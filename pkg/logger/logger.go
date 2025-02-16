package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitLogger -- init logger
func InitLogger() *zap.SugaredLogger {
	// init production encoder conf
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	// init production conf
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig = encoderCfg
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stdout"}
	// build logger
	logger, _ := cfg.Build()

	sugarLog := logger.Sugar()
	return sugarLog
}

// Close will flush log to file
func Close(l *zap.SugaredLogger) {
	_ = l.Sync()
}
