/*
Copyright 2020 Google LLC

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

package logging

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

const (
	// Default log level
	DefaultLevel = "info"

	// Text format
	FormatText = "text"
	// Colored text format
	FormatColor = "color"
	// JSON format
	FormatJSON = "json"
)

var (
	logLevel  string
	logFormat string
)

// AddFlags injects logging-related flags into the given FlagSet
func AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&logLevel, "verbosity", "v", DefaultLevel, "Log level (debug, info, warn, error, fatal, panic")
	fs.StringVar(&logFormat, "log-format", FormatColor, "Log format (text, color, json)")
}

// Configure sets the logrus logging level and formatter
func Configure() error {
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return errors.Wrap(err, "parsing log level")
	}
	logrus.SetLevel(lvl)

	var formatter logrus.Formatter
	switch logFormat {
	case FormatText:
		formatter = &logrus.TextFormatter{
			DisableColors: true,
		}
	case FormatColor:
		formatter = &logrus.TextFormatter{
			ForceColors: true,
		}
	case FormatJSON:
		formatter = &logrus.JSONFormatter{}
	default:
		return fmt.Errorf("not a valid log format: %q. Please specify one of (text, color, json)", logFormat)
	}
	logrus.SetFormatter(formatter)

	return nil
}
