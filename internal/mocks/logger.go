package mocks

import (
	"io"
	"log"

	"github.com/Rhaqim/buckt/internal/domain"
)

type NoopLogger struct{}

var _ domain.BucktLogger = (*NoopLogger)(nil)

func (NoopLogger) AddLogger(*log.Logger)                  {}
func (NoopLogger) Errorf(string, ...any)                  {}
func (NoopLogger) GetLogger() *log.Logger                 { return nil }
func (NoopLogger) Info(string)                            {}
func (NoopLogger) Infof(string, ...any)                   {}
func (NoopLogger) Warn(string)                            {}
func (NoopLogger) WrapError(string, error) error          { return nil }
func (NoopLogger) WrapErrorf(string, error, ...any) error { return nil }
func (NoopLogger) Writer() io.Writer                      { return io.Discard }
