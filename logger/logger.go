package logger

import (
	"log/syslog"
	"os"

	"github.com/spf13/viper"
	"github.com/tchap/zapext/v2/zapsyslog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New Logger
func New(v *viper.Viper) (*zap.Logger, error) {
	lvl := getLoggerLevel(v)
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        zapcore.OmitKey,
		NameKey:        "logger",
		CallerKey:      zapcore.OmitKey,
		FunctionKey:    zapcore.OmitKey,
		StacktraceKey:  "trace",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	})
	core := zapcore.NewCore(encoder, os.Stdout, lvl)

	if v.GetString("logger_syslog_addr") == "" {
		return zap.New(core), nil
	}

	v.SetDefault("logger_syslog_protocol", "udp")
	v.SetDefault("logger_syslog_app_name", "redmine_mattermost_bot")

	writer, err := syslog.Dial(
		v.GetString("logger_syslog_protocol"),
		v.GetString("logger_syslog_addr"),
		syslog.LOG_LOCAL5,
		v.GetString("logger_syslog_app_name"),
	)

	if err != nil {
		return nil, err
	}

	return zap.New(zapsyslog.NewCore(lvl, encoder, writer)), nil
}

func getLoggerLevel(v *viper.Viper) zapcore.Level {
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(v.GetString("logger_level"))); err != nil {
		return zapcore.ErrorLevel
	}

	return l
}
