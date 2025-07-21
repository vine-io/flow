package flow

import (
	"context"
	"fmt"
	"time"

	vclient "github.com/vine-io/vine/core/client"
	verrs "github.com/vine-io/vine/lib/errors"
	log "github.com/vine-io/vine/lib/logger"

	"github.com/vine-io/flow/api"
)

type LoggerOption func(options *LoggerOptions)

func WithLoggerContext(ctx *PipeSessionCtx) LoggerOption {
	return func(opts *LoggerOptions) {
		opts.ctx = ctx
	}
}

type LoggerOptions struct {
	ctx *PipeSessionCtx
}

func NewLoggerOptions(opts ...LoggerOption) LoggerOptions {
	var options LoggerOptions

	for _, opt := range opts {
		opt(&options)
	}

	return options
}

type Logger struct {
	LoggerOptions
}

func NewLogger(opts ...LoggerOption) *Logger {
	options := NewLoggerOptions(opts...)
	return &Logger{LoggerOptions: options}
}

func (lg *Logger) log(ctx context.Context, level api.TraceLevel, text string, opts ...vclient.CallOption) error {
	if len(lg.ctx.wid) == 0 {
		return nil
	}

	traceLog := &api.TraceLog{
		Level:      level,
		Wid:        lg.ctx.wid,
		InstanceId: lg.ctx.instanceId,
		Sid:        lg.ctx.sid,
		Text:       text,
		Timestamp:  time.Now().UnixNano(),
	}

	in := &api.StepTraceRequest{TraceLog: traceLog}
	opts = append(lg.ctx.c.cfg.callOptions(), opts...)
	_, err := lg.ctx.c.s.StepTrace(ctx, in, opts...)
	if err != nil {
		return verrs.FromErr(err)
	}
	return nil
}

func (lg *Logger) Trace(text string) {
	_ = lg.log(lg.ctx, api.TraceLevel_TL_TRACE, text)
	log.Trace(text)
}

func (lg *Logger) Tracef(format string, args ...any) {
	_ = lg.log(lg.ctx, api.TraceLevel_TL_TRACE, fmt.Sprintf(format, args...))
	log.Tracef(format, args...)
}

func (lg *Logger) Debug(text string) {
	_ = lg.log(lg.ctx, api.TraceLevel_TL_DEBUG, text)
	log.Debug(text)
}

func (lg *Logger) Debugf(format string, args ...any) {
	_ = lg.log(lg.ctx, api.TraceLevel_TL_DEBUG, fmt.Sprintf(format, args...))
	log.Debugf(format, args...)
}

func (lg *Logger) Info(text string) {
	_ = lg.log(lg.ctx, api.TraceLevel_TL_INFO, text)
	log.Info(text)
}

func (lg *Logger) Infof(format string, args ...any) {
	_ = lg.log(lg.ctx, api.TraceLevel_TL_INFO, fmt.Sprintf(format, args...))
	log.Infof(format, args...)
}

func (lg *Logger) Warn(text string) {
	_ = lg.log(lg.ctx, api.TraceLevel_TL_WARN, text)
	log.Warn(text)
}

func (lg *Logger) Warnf(format string, args ...any) {
	_ = lg.log(lg.ctx, api.TraceLevel_TL_WARN, fmt.Sprintf(format, args...))
	log.Warnf(format, args...)
}

func (lg *Logger) Error(text string) {
	_ = lg.log(lg.ctx, api.TraceLevel_TL_ERROR, text)
	log.Error(text)
}

func (lg *Logger) Errorf(format string, args ...any) {
	_ = lg.log(lg.ctx, api.TraceLevel_TL_ERROR, fmt.Sprintf(format, args...))
	log.Errorf(format, args...)
}
