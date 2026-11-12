package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/meshery/meshkit/logger"
	"github.com/sirupsen/logrus"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

const appName = "meshery-mcp-server"

// Set at build time via -ldflags "-X main.version=... -X main.commitsha=...".
var (
	version   = "v0.0.0-unknown"
	commitsha = "unknown"
)

func main() {
	log, err := logger.New(appName, logger.Options{
		Format:   logger.SyslogLogFormat,
		LogLevel: int(logrus.InfoLevel),
		Output:   os.Stderr, // stdout is reserved for the MCP protocol
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if err := run(log); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func run(log logger.Handler) error {
	configPath := flag.String("config", "", "path to a YAML config file (optional)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	log.SetLevel(logrusLevel(cfg.Log.Level))

	log.Infof("starting %s version=%s commit=%s meshery_url=%s token_configured=%t",
		appName, version, commitsha, cfg.Meshery.URL, cfg.Meshery.Token != "")

	client := meshery.NewClient(
		cfg.Meshery.URL,
		string(cfg.Meshery.Token),
		meshery.WithKeychainScope(cfg.Meshery.KeychainScope),
		meshery.WithMaxConcurrentCalls(cfg.Meshery.MaxConcurrentCalls),
		meshery.WithMaxRetries(cfg.Meshery.MaxRetries),
		meshery.WithLogger(log),
	)
	log.Infof("meshery cloud client ready: keychain_scope=%q max_concurrent_calls=%d max_retries=%d",
		cfg.Meshery.KeychainScope, cfg.Meshery.MaxConcurrentCalls, cfg.Meshery.MaxRetries)

	// TODO: create MCP server; serve over stdio using `client`.
	_ = client
	return nil
}

func logrusLevel(level string) logrus.Level {
	switch level {
	case config.LogLevelDebug:
		return logrus.DebugLevel
	case config.LogLevelWarn:
		return logrus.WarnLevel
	case config.LogLevelError:
		return logrus.ErrorLevel
	default:
		return logrus.InfoLevel
	}
}
