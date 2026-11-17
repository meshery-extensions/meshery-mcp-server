package main

import (
	"context"
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

	client, err := meshery.New(meshery.Config{
		BaseURL:            cfg.Meshery.URL,
		Token:              string(cfg.Meshery.Token),
		Timeout:            cfg.Meshery.Timeout,
		MaxRetries:         cfg.Meshery.MaxRetries,
		MaxConcurrentCalls: cfg.Meshery.MaxConcurrentCalls,
		KeychainScope:      cfg.Meshery.KeychainScope,
		AuthMode:           meshery.AuthCookie,
	})
	if err != nil {
		return err
	}
	log.Infof("meshery client ready: url=%s token_configured=%t keychain_scope=%q max_concurrent_calls=%d max_retries=%d timeout=%s",
		cfg.Meshery.URL, cfg.Meshery.Token != "", cfg.Meshery.KeychainScope,
		cfg.Meshery.MaxConcurrentCalls, cfg.Meshery.MaxRetries, cfg.Meshery.Timeout)

	// A health check is advisory: the MCP server can still start so tools can
	// report the failure themselves.
	pingCtx, cancel := context.WithTimeout(context.Background(), cfg.Meshery.Timeout)
	defer cancel()
	if version, err := client.Ping(pingCtx); err != nil {
		log.Warnf("meshery health check failed: %v", err)
	} else {
		log.Infof("meshery reachable: build=%q commitsha=%q", version.Build, version.Commitsha)
	}

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
