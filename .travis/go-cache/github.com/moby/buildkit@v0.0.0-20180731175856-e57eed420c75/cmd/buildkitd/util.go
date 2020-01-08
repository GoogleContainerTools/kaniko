package main

import (
	"strconv"
	"strings"
)

// parseBoolOrAuto returns (nil, nil) if s is "auto"
func parseBoolOrAuto(s string) (*bool, error) {
	if s == "" || strings.ToLower(s) == "auto" {
		return nil, nil
	}
	b, err := strconv.ParseBool(s)
	return &b, err
}
