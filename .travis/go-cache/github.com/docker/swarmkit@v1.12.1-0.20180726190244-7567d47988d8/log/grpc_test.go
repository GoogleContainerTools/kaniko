package log

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGRPCLogrusLevelTranslation(t *testing.T) {
	logger := logrus.New()
	wrapped := logrusWrapper{Entry: logrus.NewEntry(logger)}
	for _, tc := range []struct {
		level     logrus.Level
		grpcLevel int
	}{
		{
			level:     logrus.InfoLevel,
			grpcLevel: 0,
		},
		{
			level:     logrus.WarnLevel,
			grpcLevel: 1,
		},
		{
			level:     logrus.ErrorLevel,
			grpcLevel: 2,
		},
		{
			level:     logrus.FatalLevel,
			grpcLevel: 3,
		},
		// these don't translate to valid grpc log levels, but should still work
		{
			level:     logrus.DebugLevel,
			grpcLevel: -1,
		},
		{
			level:     logrus.PanicLevel,
			grpcLevel: 4,
		},
	} {
		logger.SetLevel(tc.level)
		for i := -1; i < 5; i++ {
			verbosityAtLeastI := wrapped.V(i)
			require.Equal(t, i <= tc.grpcLevel, verbosityAtLeastI,
				"Is verbosity at least %d? Logrus level at %v", i, tc.level)
		}
	}

	// these values should also always work, even though they're not valid grpc log values
	logrus.SetLevel(logrus.DebugLevel)
	require.True(t, wrapped.V(-100))

	logrus.SetLevel(logrus.PanicLevel)
	require.False(t, wrapped.V(100))
}
