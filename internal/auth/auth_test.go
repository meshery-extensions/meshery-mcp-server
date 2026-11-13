package auth

import (
	"context"
	"testing"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

func TestResolver(t *testing.T) {
	t.Run("no token attaches nothing", func(t *testing.T) {
		if resolve := Resolver(config.AuthConfig{}); resolve != nil {
			t.Error("want nil resolver without a token")
		}
	})

	t.Run("token is attached to every request", func(t *testing.T) {
		resolve := Resolver(config.AuthConfig{Token: "abc"})
		if resolve == nil {
			t.Fatal("want a resolver for a token")
		}
		for range 2 {
			ctx, err := resolve(context.Background(), nil)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if got, ok := meshery.TokenFromContext(ctx); !ok || got != "abc" {
				t.Errorf("token = (%q, %v), want (%q, true)", got, ok, "abc")
			}
		}
	})
}
