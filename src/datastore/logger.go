package datastore

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm/logger"
)

var dataStoreTracer = func(ctx context.Context) *logrus.Entry {
	return logrus.WithField("comp", "datastore").WithContext(ctx)
}

type Logger struct {
	slowThreshold time.Duration
}

func (logger Logger) LogMode(_ logger.LogLevel) logger.Interface {
	return logger
}

func (logger Logger) Info(ctx context.Context, format string, msg ...interface{}) {
	dataStoreTracer(ctx).Infof(format, msg...)
}
func (logger Logger) Warn(ctx context.Context, format string, msg ...interface{}) {
	dataStoreTracer(ctx).Warningf(format, msg...)
}
func (logger Logger) Error(ctx context.Context, format string, msg ...interface{}) {
	dataStoreTracer(ctx).Errorf(format, msg...)
}
func (logger Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, _ := fc()

	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		dataStoreTracer(ctx).Errorf("SQL failed: %s, err=%s", sql, err.Error())
	case elapsed > logger.slowThreshold:
		dataStoreTracer(ctx).Warningf("[SLOW SQL] - %s | %s", elapsed.String(), sql)
	default:
		dataStoreTracer(ctx).Debugf("SQL - %s | %s", elapsed.String(), sql)
	}
}
