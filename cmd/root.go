package cmd

import (
	"os"

	isatty "github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	var logger *zap.Logger
	var err error
	errorPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})

	infoPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel && lvl >= zapcore.InfoLevel
	})

	consoleErrors := zapcore.Lock(os.Stderr)
	consoleInfo := zapcore.Lock(os.Stdout)

	var consoleEncoder zapcore.Encoder
	if isatty.IsTerminal(os.Stdout.Fd()) {
		encoderConf := zap.NewDevelopmentEncoderConfig()
		encoderConf.EncodeLevel = zapcore.CapitalColorLevelEncoder
		consoleEncoder = zapcore.NewConsoleEncoder(encoderConf)
	} else {
		encoderConf := zap.NewProductionEncoderConfig()
		encoderConf.MessageKey = "message"
		encoderConf.EncodeTime = zapcore.TimeEncoder(zapcore.ISO8601TimeEncoder)
		consoleEncoder = zapcore.NewJSONEncoder(encoderConf)
	}

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleErrors, errorPriority),
		zapcore.NewCore(consoleEncoder, consoleInfo, infoPriority),
	)

	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}

	stackTraceEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl > zapcore.ErrorLevel
	})
	logger = zap.New(core, zap.Fields(zap.String("host", host)), zap.AddStacktrace(stackTraceEnabler))

	if err != nil {
		zap.S().Fatalw("Failed to create logger", "error", err)
	}

	zap.ReplaceGlobals(logger.Named("app"))
	zap.RedirectStdLog(logger.Named("stdlog"))
}

// GetCommands returns the set of available commands in the application
func GetCommands() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ingestor",
		Short: "Ingestion service",
	}

	rootCmd.AddCommand(createIngestCommand())
	rootCmd.AddCommand(createUpdateCommand())

	return rootCmd
}
