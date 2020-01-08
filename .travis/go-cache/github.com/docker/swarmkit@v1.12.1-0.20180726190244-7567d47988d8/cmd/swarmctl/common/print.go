package common

import (
	"fmt"
	"io"
	"strings"

	"github.com/dustin/go-humanize"
	gogotypes "github.com/gogo/protobuf/types"
)

// PrintHeader prints a nice little header.
func PrintHeader(w io.Writer, columns ...string) {
	underline := make([]string, len(columns))
	for i := range underline {
		underline[i] = strings.Repeat("-", len(columns[i]))
	}
	fmt.Fprintf(w, "%s\n", strings.Join(columns, "\t"))
	fmt.Fprintf(w, "%s\n", strings.Join(underline, "\t"))
}

// FprintfIfNotEmpty prints only if `s` is not empty.
//
// NOTE(stevvooe): Not even remotely a printf function.. doesn't take args.
func FprintfIfNotEmpty(w io.Writer, format string, v interface{}) {
	if v != nil && v != "" {
		fmt.Fprintf(w, format, v)
	}
}

// TimestampAgo returns a relative time string from a timestamp (e.g. "12 seconds ago").
func TimestampAgo(ts *gogotypes.Timestamp) string {
	if ts == nil {
		return ""
	}
	t, err := gogotypes.TimestampFromProto(ts)
	if err != nil {
		panic(err)
	}
	return humanize.Time(t)
}
