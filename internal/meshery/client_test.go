package meshery

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestCredentialsFromContext(t *testing.T) {
	remote := Credentials{Provider: "Meshery", Token: "abc"}
	local := Credentials{Provider: "Local"}

	tests := []struct {
		name   string
		ctx    context.Context
		want   Credentials
		wantOK bool
	}{
		{name: "none", ctx: context.Background()},
		{name: "empty", ctx: WithCredentials(context.Background(), Credentials{})},
		{name: "provider and token", ctx: WithCredentials(context.Background(), remote), want: remote, wantOK: true},
		{name: "provider without token", ctx: WithCredentials(context.Background(), local), want: local, wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := CredentialsFromContext(tt.ctx)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("CredentialsFromContext = (%v, %v), want (%v, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestTokenIsNotPrinted(t *testing.T) {
	creds := Credentials{Provider: "Meshery", Token: "super-secret"}
	ctx := WithCredentials(context.Background(), creds)

	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		for name, value := range map[string]any{"context": ctx, "credentials": creds, "pointer": &creds} {
			out := fmt.Sprintf(format, value)
			if strings.Contains(out, "super-secret") {
				t.Errorf("%s with %s leaked the token: %s", name, format, out)
			}
		}
	}
	if out := fmt.Sprint(creds); !strings.Contains(out, "Meshery") {
		t.Errorf("Credentials print as %q, want the provider shown", out)
	}
}
