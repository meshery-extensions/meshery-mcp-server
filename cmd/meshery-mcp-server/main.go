package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/meshery/meshkit/logger"

	"github.com/meshery-extensions/meshery-mcp-server/internal/auth"
	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
	"github.com/meshery-extensions/meshery-mcp-server/internal/logging"
	mcpserver "github.com/meshery-extensions/meshery-mcp-server/internal/mcp-server"
	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

const appName = "meshery-mcp-server"

// Set at build time via -ldflags "-X main.version=... -X main.commitsha=...".
var (
	version   = "v0.0.0-unknown"
	commitsha = "unknown"
)

func main() {
	// Until the config is loaded, log as text at info level. Logs go to
	// stderr: over stdio, stdout carries the MCP protocol.
	boot, _, err := logging.New(appName, config.LogFormatText, config.LogLevelInfo, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if log, err := run(boot); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

// run returns the logger to report its error with: the boot logger if the
// config could not be loaded, the configured logger otherwise.
func run(boot logger.Handler) (logger.Handler, error) {
	configPath := flag.String("config", "", "path to a YAML config file (optional)")
	transport := flag.String("transport", "", "stdio or http (default stdio; env MESHERY_MCP_TRANSPORT)")
	host := flag.String("host", "", "http only: loopback address to listen on (default 127.0.0.1; env MESHERY_MCP_HOST)")
	port := flag.Int("port", 0, "http only: port to listen on (default 8080; env MESHERY_MCP_PORT)")
	flag.Parse()

	// Only flags given on the command line override the environment and file.
	var flags config.Flags
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "transport":
			flags.Transport = transport
		case "host":
			flags.Host = host
		case "port":
			flags.Port = port
		}
	})

	cfg, err := config.Load(*configPath, flags)
	if err != nil {
		return boot, err
	}
	log, sdkLog, err := logging.New(appName, cfg.Log.Format, cfg.Log.Level, os.Stderr)
	if err != nil {
		return boot, err
	}

	log.Infof("starting %s version=%s commit=%s meshery_url=%s token_configured=%t transport=%s",
		appName, version, commitsha, cfg.Meshery.RedactedURL(), cfg.Auth.Token != "", cfg.Transport.Mode)

	client, err := meshery.New(meshery.Config{
		BaseURL:            cfg.Meshery.URL,
		Token:              string(cfg.Auth.Token),
		Timeout:            cfg.Meshery.Timeout,
		MaxRetries:         cfg.Meshery.MaxRetries,
		MaxConcurrentCalls: cfg.Meshery.MaxConcurrentCalls,
		KeychainScope:      cfg.Meshery.KeychainScope,
		AuthMode:           meshery.AuthCookie,
	})
	if err != nil {
		return boot, err
	}
	log.Infof("meshery client ready: url=%s token_configured=%t keychain_scope=%q max_concurrent_calls=%d max_retries=%d timeout=%s",
		cfg.Meshery.URL, cfg.Auth.Token != "", cfg.Meshery.KeychainScope,
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		// After the first signal, restore default handling so a second
		// Ctrl+C exits immediately instead of waiting for shutdown.
		<-ctx.Done()
		stop()
	}()

	opts := mcpserver.Options{
		Name:        appName,
		Version:     version,
		Logger:      sdkLog,
		Registry:    meshery.NewRegistry(client),
		Credentials: auth.Resolver(cfg.Auth),
	}

	switch cfg.Transport.Mode {
	case config.TransportHTTP:
		err = mcpserver.RunHTTP(ctx, opts, mcpserver.HTTPOptions{
			Addr:      net.JoinHostPort(cfg.Transport.Host, strconv.Itoa(cfg.Transport.Port)),
			AccessKey: string(cfg.Transport.AccessKey),
		})
	default:
		err = mcpserver.RunStdio(ctx, opts)
	}
	if err != nil {
		return log, err
	}
	log.Info("stopped")
	return log, nil
}
