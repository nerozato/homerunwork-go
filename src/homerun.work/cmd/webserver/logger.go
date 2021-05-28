package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/natefinch/lumberjack.v2"
)

//ZapConfig : zap logger configuration
const ZapConfig = `
{
	"development": true,
	"encoderConfig": {
		"callerEncoder": "short",
		"callerKey": "caller",
		"levelEncoder": "lowercase",
		"levelKey": "lvl",
		"messageKey": "msg",
		"stacktraceKey": "stack",
		"timeEncoder": "rfc3339nano",
		"timeKey": "ts"
	},
	"encoding": "json",
	"errorOutputPaths": [
		"stderr"
	],
	"outputPaths": [
		"stdout"
	]
}
`

//maximum log size of 10MB
const logMaxSize = 10

//name of the log file
const logFileName = "log/homerun.log"

//scheme used for lumberjack
const logLumberJackScheme = "lumberjack"

//Logger : logger
type Logger struct {
	logger *zap.SugaredLogger
	ctx    context.Context
}

//With : wrap adding a parameter
func (l *Logger) With(args ...interface{}) *Logger {
	logger := l.logger.With(args...)
	return &Logger{
		logger: logger,
		ctx:    l.ctx,
	}
}

//Debugf : wrap the debugf call
func (l *Logger) Debugf(msg string, args ...interface{}) {
	l.logger.Debugf(msg, args...)
}

//Debugw : wrap the debugw call
func (l *Logger) Debugw(msg string, args ...interface{}) {
	l.logger.Debugw(msg, args...)
}

//Infof : wrap the infof call
func (l *Logger) Infof(msg string, args ...interface{}) {
	l.logger.Infof(msg, args...)
}

//Infow : wrap the infow call
func (l *Logger) Infow(msg string, args ...interface{}) {
	l.logger.Infow(msg, args...)
}

//Warnf : wrap the warnf call
func (l *Logger) Warnf(msg string, args ...interface{}) {
	l.logger.Warnf(msg, args...)
	AddCtxStatsCount(l.ctx, ServerStatLogWarnings, 1)
}

//Warnw : wrap the warnw call
func (l *Logger) Warnw(msg string, args ...interface{}) {
	l.logger.Warnw(msg, args...)
	AddCtxStatsCount(l.ctx, ServerStatLogWarnings, 1)
}

//Errorf : wrap the errorf call
func (l *Logger) Errorf(msg string, args ...interface{}) {
	l.logger.Errorf(msg, args...)
	AddCtxStatsCount(l.ctx, ServerStatLogErrors, 1)
}

//Errorw : wrap the errorw call
func (l *Logger) Errorw(msg string, args ...interface{}) {
	l.logger.Errorw(msg, args...)
	AddCtxStatsCount(l.ctx, ServerStatLogErrors, 1)
}

//Panicf : wrap the panicf call
func (l *Logger) Panicf(msg string, args ...interface{}) {
	l.logger.Panicf(msg, args...)
	AddCtxStatsCount(l.ctx, ServerStatLogPanics, 1)
}

//Panicw : wrap the panicw call
func (l *Logger) Panicw(msg string, args ...interface{}) {
	l.logger.Panicw(msg, args...)
	AddCtxStatsCount(l.ctx, ServerStatLogPanics, 1)
}

//wrapper for the lumberjack logger to be used as a sink
type lumberjackSink struct {
	*lumberjack.Logger
}

//required for a zap sink
func (l lumberjackSink) Sync() error {
	return nil
}

//InitLogger : initialize the logger based on the configuration
func InitLogger(ctx context.Context, requestID string) (*Logger, error) {
	//configure the logger
	var cfg zap.Config
	err := json.Unmarshal([]byte(ZapConfig), &cfg)
	if err != nil {
		return nil, errors.Wrap(err, "json unmarshal log config")
	}
	level := zap.NewAtomicLevel()
	level.UnmarshalText([]byte(GetLogLevel()))
	cfg.Level = level
	cfg.Development = GetLogDevEnable()

	//enhance with the request id
	if requestID != "" {
		cfg.InitialFields = map[string]interface{}{
			"request_id": requestID,
		}
	}

	//register lumberjack as a sink
	if GetLogFileEnable() {
		zap.RegisterSink(logLumberJackScheme, func(u *url.URL) (zap.Sink, error) {
			return lumberjackSink{
				Logger: &lumberjack.Logger{
					Filename: u.Opaque,
					MaxSize:  logMaxSize,
				},
			}, nil
		})

		//add lumberjack as an output
		logLumberJackURI := fmt.Sprintf("%s:%s", logLumberJackScheme, logFileName)
		cfg.OutputPaths = append(cfg.OutputPaths, logLumberJackURI)
		cfg.ErrorOutputPaths = append(cfg.ErrorOutputPaths, logLumberJackURI)
	}

	//build the logger
	l, err := cfg.Build()
	if err != nil {
		return nil, errors.Wrap(err, "build logger")
	}

	//use as the global logger
	zap.ReplaceGlobals(l)
	logger := &Logger{
		logger: zap.S(),
		ctx:    ctx,
	}
	return logger, nil
}

//GetLogger : retrieve the logger based on the context
func GetLogger(ctx context.Context, args ...interface{}) (context.Context, *Logger) {
	//check if a context-specific logger is required
	if ctx == nil {
		logger := &Logger{
			logger: zap.S(),
			ctx:    ctx,
		}
		return ctx, logger
	}

	//check if a logger is already available
	logger := GetCtxLogger(ctx)
	if logger != nil {
		return ctx, logger
	}

	//check if a request id should be associated with the logger
	requestID := GetCtxRequestID(ctx)
	if requestID == "" {
		logger := &Logger{
			logger: zap.S(),
			ctx:    ctx,
		}
		return ctx, logger
	}

	//store the logger with the associated request id
	zap := zap.S().With("requestId", requestID).With(args...)
	logger = &Logger{
		logger: zap,
		ctx:    ctx,
	}
	ctx = SetCtxLogger(ctx, logger)
	return ctx, logger
}

//SetLoggerUserID : associate a user id with the logger
func SetLoggerUserID(ctx context.Context, userID *uuid.UUID) context.Context {
	ctx, logger := GetLogger(ctx)
	logger = logger.With("userId", userID.String())
	ctx = SetCtxLogger(ctx, logger)
	return ctx
}
