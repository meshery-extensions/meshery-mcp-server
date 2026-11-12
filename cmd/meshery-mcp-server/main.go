package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/meshery/meshkit/logger"
	"github.com/sirupsen/logrus"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
	"github.com/meshery-extensions/meshery-mcp-server/internal/prompts"
	"github.com/meshery-extensions/meshery-mcp-server/internal/resources"
	"github.com/meshery-extensions/meshery-mcp-server/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	transport := flag.String("transport", "stdio", "MCP transport: stdio or http")
	port := flag.Int("port", 8080, "TCP port for the http transport")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	log.SetLevel(logrusLevel(cfg.Log.Level))

	log.Infof("starting %s version=%s commit=%s meshery_url=%s token_configured=%t",
		appName, version, commitsha, cfg.Meshery.URL, cfg.Meshery.Token != "")

	// Build the go-sdk MCP server and register tools, resources, prompts.
	server := mcp.NewServer(
		&mcp.Implementation{Name: "meshery-mcp-server", Version: version},
		nil,
	)
	if err := tools.Register(server); err != nil {
		return fmt.Errorf("registering tools: %w", err)
	}
	if err := resources.Register(server); err != nil {
		return fmt.Errorf("registering resources: %w", err)
	}
	if err := prompts.Register(server); err != nil {
		return fmt.Errorf("registering prompts: %w", err)
	}

	log.Infof("meshery MCP server ready: transport=%s protocol=2026-07-28 (negotiates down to 2025-11-25)",
		*transport)

	switch *transport {
	case "stdio":
		// stdout is reserved for the MCP protocol; logs go to stderr.
		return server.Run(context.Background(), &mcp.StdioTransport{})
	case "http":
		handler := mcp.NewStreamableHTTPHandler(
			func(*http.Request) *mcp.Server { return server },
			&mcp.StreamableHTTPOptions{Stateless: true},
		)
		addr := fmt.Sprintf("127.0.0.1:%d", *port)
		log.Infof("listening on http://%s", addr)
		srv := &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		}
		return srv.ListenAndServe()
	default:
		return fmt.Errorf("unknown transport %q: use stdio or http", *transport)
	}
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
