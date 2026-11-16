package mcpserver

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	HTTPPath = "/mcp"

	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 5 * time.Second
)

type HTTPOptions struct {
	Addr string // host:port
}

// RunHTTP returns nil when ctx is cancelled. MCP authorization goes in front
// of the handler; config refuses http without it.
func RunHTTP(ctx context.Context, opts Options, httpOpts HTTPOptions) error {
	server, err := newServer(opts)
	if err != nil {
		return err
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", httpOpts.Addr)
	if err != nil {
		return ErrListen(err, httpOpts.Addr)
	}
	return serveHTTP(ctx, server, ln, opts.Logger)
}

func newHTTPHandler(server *mcp.Server, logger *slog.Logger) http.Handler {
	streamable := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{
			// Protocol 2026-07-28 has no sessions and the SDK serves it over
			// HTTP only statelessly; older clients are served the same way.
			Stateless:                    true,
			PropagateRequestCancellation: true,
			Logger:                       logger,
		},
	)

	// The SDK handler rejects non-localhost Host headers (DNS rebinding).
	handler := http.NewCrossOriginProtection().Handler(noProxyBuffering(streamable))

	mux := http.NewServeMux()
	mux.Handle(HTTPPath, handler)
	return mux
}

// noProxyBuffering stops proxies such as nginx from holding back SSE events.
func noProxyBuffering(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Accel-Buffering", "no")
		next.ServeHTTP(w, r)
	})
}

func serveHTTP(ctx context.Context, server *mcp.Server, ln net.Listener, logger *slog.Logger) error {
	srv := &http.Server{
		Handler:           newHTTPHandler(server, logger),
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}
	if logger != nil {
		srv.ErrorLog = slog.NewLogLogger(logger.Handler(), slog.LevelError)
		logger.Info("serving MCP over Streamable HTTP", "url", "http://"+ln.Addr().String()+HTTPPath)
	}

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close()
		return ErrShutdownHTTP(err)
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
