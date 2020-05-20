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
)

const (
	// Default log level
	DefaultLevel = "info"
	// Default timestamp in logs
	DefaultLogTimestamp = false

	// Text format
	FormatText = "text"
	// Colored text format
	FormatColor = "color"
	// JSON format
	FormatJSON = "json"
)

// Configure sets the logrus logging level and formatter
func Configure(level, format string, logTimestamp bool) error {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return errors.Wrap(err, "parsing log level")
	}
	logrus.SetLevel(lvl)

	var formatter logrus.Formatter
	switch format {
	case FormatText:
		formatter = &logrus.TextFormatter{
			DisableColors: true,
			FullTimestamp: logTimestamp,
		}
	case FormatColor:
		formatter = &logrus.TextFormatter{
			ForceColors:   true,
			FullTimestamp: logTimestamp,
		}
	case FormatJSON:
		formatter = &logrus.JSONFormatter{}
	default:
		return fmt.Errorf("not a valid log format: %q. Please specify one of (text, color, json)", format)
	}
	logrus.SetFormatter(formatter)

	return nil
}
