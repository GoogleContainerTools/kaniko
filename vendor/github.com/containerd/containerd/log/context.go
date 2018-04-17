<<<<<<< HEAD
/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

=======
>>>>>>> WIP: set the docker default seccomp profile in the executor process.
package log

import (
	"context"
<<<<<<< HEAD
	"sync/atomic"
=======
	"path"
>>>>>>> WIP: set the docker default seccomp profile in the executor process.

	"github.com/sirupsen/logrus"
)

var (
	// G is an alias for GetLogger.
	//
	// We may want to define this locally to a package to get package tagged log
	// messages.
	G = GetLogger

	// L is an alias for the the standard logger.
	L = logrus.NewEntry(logrus.StandardLogger())
)

type (
	loggerKey struct{}
<<<<<<< HEAD
)

// TraceLevel is the log level for tracing. Trace level is lower than debug level,
// and is usually used to trace detailed behavior of the program.
const TraceLevel = logrus.Level(uint32(logrus.DebugLevel + 1))

// ParseLevel takes a string level and returns the Logrus log level constant.
// It supports trace level.
func ParseLevel(lvl string) (logrus.Level, error) {
	if lvl == "trace" {
		return TraceLevel, nil
	}
	return logrus.ParseLevel(lvl)
}

=======
	moduleKey struct{}
)

>>>>>>> WIP: set the docker default seccomp profile in the executor process.
// WithLogger returns a new context with the provided logger. Use in
// combination with logger.WithField(s) for great effect.
func WithLogger(ctx context.Context, logger *logrus.Entry) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// GetLogger retrieves the current logger from the context. If no logger is
// available, the default logger is returned.
func GetLogger(ctx context.Context) *logrus.Entry {
	logger := ctx.Value(loggerKey{})

	if logger == nil {
		return L
	}

	return logger.(*logrus.Entry)
}

<<<<<<< HEAD
// Trace logs a message at level Trace with the log entry passed-in.
func Trace(e *logrus.Entry, args ...interface{}) {
	level := logrus.Level(atomic.LoadUint32((*uint32)(&e.Logger.Level)))
	if level >= TraceLevel {
		e.Debug(args...)
	}
}

// Tracef logs a message at level Trace with the log entry passed-in.
func Tracef(e *logrus.Entry, format string, args ...interface{}) {
	level := logrus.Level(atomic.LoadUint32((*uint32)(&e.Logger.Level)))
	if level >= TraceLevel {
		e.Debugf(format, args...)
	}
=======
// WithModule adds the module to the context, appending it with a slash if a
// module already exists. A module is just an roughly correlated defined by the
// call tree for a given context.
//
// As an example, we might have a "node" module already part of a context. If
// this function is called with "tls", the new value of module will be
// "node/tls".
//
// Modules represent the call path. If the new module and last module are the
// same, a new module entry will not be created. If the new module and old
// older module are the same but separated by other modules, the cycle will be
// represented by the module path.
func WithModule(ctx context.Context, module string) context.Context {
	parent := GetModulePath(ctx)

	if parent != "" {
		// don't re-append module when module is the same.
		if path.Base(parent) == module {
			return ctx
		}

		module = path.Join(parent, module)
	}

	ctx = WithLogger(ctx, GetLogger(ctx).WithField("module", module))
	return context.WithValue(ctx, moduleKey{}, module)
}

// GetModulePath returns the module path for the provided context. If no module
// is set, an empty string is returned.
func GetModulePath(ctx context.Context) string {
	module := ctx.Value(moduleKey{})
	if module == nil {
		return ""
	}

	return module.(string)
>>>>>>> WIP: set the docker default seccomp profile in the executor process.
}
