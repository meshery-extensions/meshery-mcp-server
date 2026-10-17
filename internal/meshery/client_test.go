package meshery

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestTokenFromContext(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		want   string
		wantOK bool
	}{
		{name: "no token", ctx: context.Background()},
		{name: "empty token", ctx: WithToken(context.Background(), "")},
		{name: "token", ctx: WithToken(context.Background(), "abc"), want: "abc", wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := TokenFromContext(tt.ctx)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("TokenFromContext = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestTokenIsNotPrinted(t *testing.T) {
	ctx := WithToken(context.Background(), "super-secret")
	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		if out := fmt.Sprintf(format, ctx); strings.Contains(out, "super-secret") {
			t.Errorf("%s leaked the token: %s", format, out)
		}
	}
}
