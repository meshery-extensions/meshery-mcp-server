package auth

import (
	"context"
	"testing"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

func TestResolver(t *testing.T) {
	t.Run("no credentials attaches nothing", func(t *testing.T) {
		if resolve := Resolver(config.StdioConfig{}); resolve != nil {
			t.Error("want nil resolver without a provider or token")
		}
	})

	tests := []struct {
		name string
		cfg  config.StdioConfig
		want meshery.Credentials
	}{
		{name: "provider and token are attached",
			cfg:  config.StdioConfig{Provider: "Meshery", Token: "abc"},
			want: meshery.Credentials{Provider: "Meshery", Token: "abc"}},
		{name: "provider without a token is attached",
			cfg:  config.StdioConfig{Provider: "Local"},
			want: meshery.Credentials{Provider: "Local"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolve := Resolver(tt.cfg)
			if resolve == nil {
				t.Fatal("want a resolver")
			}
			for range 2 {
				ctx, err := resolve(context.Background(), nil)
				if err != nil {
					t.Fatalf("resolve: %v", err)
				}
				if got, ok := meshery.CredentialsFromContext(ctx); !ok || got != tt.want {
					t.Errorf("credentials = (%v, %v), want (%v, true)", got, ok, tt.want)
				}
			}
		})
	}
}
