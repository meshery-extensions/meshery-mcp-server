package meshery

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/meshery/meshkit/errors"

	"github.com/meshery-extensions/meshery-mcp-server/internal/tools"
)

// A stub must fail rather than return data that looks real.
func TestRegistry_NotImplemented(t *testing.T) {
	r := NewRegistry(&Client{})
	ctx := context.Background()

	calls := map[string]func() error{
		"SearchComponents": func() error {
			_, err := r.SearchComponents(ctx, "pod", "", tools.Page{Size: 25})
			return err
		},
		"GetComponent": func() error {
			_, err := r.GetComponent(ctx, uuid.Must(uuid.NewV4()))
			return err
		},
		"ListModels": func() error {
			_, err := r.ListModels(ctx, "", tools.Page{Size: 25})
			return err
		},
		"GetModel": func() error {
			_, err := r.GetModel(ctx, uuid.Must(uuid.NewV4()))
			return err
		},
		"ListRelationships": func() error {
			_, err := r.ListRelationships(ctx, "", tools.Page{Size: 25})
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			if err := call(); errors.GetCode(err) != ErrNotImplementedCode {
				t.Errorf("err = %v, want code %s", err, ErrNotImplementedCode)
			}
		})
	}
}
