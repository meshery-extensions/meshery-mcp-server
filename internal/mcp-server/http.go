package mcpserver

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// HTTPPath is where the Streamable HTTP endpoint is served.
	HTTPPath = "/mcp"

	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 5 * time.Second
)

// HTTPOptions configures the HTTP transport.
type HTTPOptions struct {
	// Addr is host:port to listen on. It must be a loopback address; config
	// enforces that.
	Addr string
	// AccessKey is the bearer token every request must carry. It must not
	// be empty: loopback stops other machines, not other local processes.
	AccessKey string
}

// RunHTTP serves Streamable HTTP until ctx is cancelled, which is a clean
// shutdown and returns nil.
//
// Every request must send "Authorization: Bearer <AccessKey>". The handler
// is stateless, rejects cross-origin browser requests, and rejects requests
// whose Host header is not localhost (DNS rebinding protection).
func RunHTTP(ctx context.Context, opts Options, httpOpts HTTPOptions) error {
	if httpOpts.AccessKey == "" {
		return ErrNoAccessKey()
	}
	server, err := newServer(opts)
	if err != nil {
		return err
	}

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", httpOpts.Addr)
	if err != nil {
		return ErrListen(err, httpOpts.Addr)
	}
	return serveHTTP(ctx, server, ln, httpOpts.AccessKey, opts.Logger)
}

func newHTTPHandler(server *mcp.Server, accessKey string, logger *slog.Logger) http.Handler {
	streamable := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{
			// No sessions: each request stands alone. Protocol 2026-07-28
			// has no sessions, and the SDK serves it over HTTP only in
			// stateless mode; 2025-era clients are served the same way.
			// Shutdown never waits on long-lived streams.
			Stateless:                    true,
			PropagateRequestCancellation: true,
			Logger:                       logger,
		},
	)

	// Outermost first: reject browser cross-origin requests, then requests
	// without the access key. The SDK handler then rejects non-localhost
	// Host headers.
	handler := http.NewCrossOriginProtection().Handler(requireAccessKey(accessKey, noProxyBuffering(streamable)))

	mux := http.NewServeMux()
	mux.Handle(HTTPPath, handler)
	return mux
}

// requireAccessKey rejects requests that do not carry
// "Authorization: Bearer <key>", comparing the key in constant time.
func requireAccessKey(key string, next http.Handler) http.Handler {
	want := []byte(key)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") ||
			subtle.ConstantTimeCompare([]byte(strings.TrimSpace(token)), want) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="meshery-mcp-server"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// noProxyBuffering asks reverse proxies such as nginx not to buffer
// responses, so SSE events reach the client as they are sent. The spec asks
// for this header on SSE responses; on JSON responses it has no effect.
func noProxyBuffering(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Accel-Buffering", "no")
		next.ServeHTTP(w, r)
	})
}

func serveHTTP(ctx context.Context, server *mcp.Server, ln net.Listener, accessKey string, logger *slog.Logger) error {
	srv := &http.Server{
		Handler:           newHTTPHandler(server, accessKey, logger),
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
