package meshery

import (
	"context"
	"testing"

	"github.com/meshery/meshkit/errors"
)

// Until a method calls Meshery, it must fail rather than return data that
// looks real.
func TestRegistry_NotImplemented(t *testing.T) {
	r := NewRegistry(&Client{})
	ctx := context.Background()

	calls := map[string]func() error{
		"SearchComponents": func() error {
			_, err := r.SearchComponents(ctx, "pod", "", "")
			return err
		},
		"GetComponent": func() error {
			_, err := r.GetComponent(ctx, "c1")
			return err
		},
		"ListModels": func() error {
			_, err := r.ListModels(ctx, "", 0, 25)
			return err
		},
		"GetModel": func() error {
			_, err := r.GetModel(ctx, "m1")
			return err
		},
		"ListRelationships": func() error {
			_, err := r.ListRelationships(ctx, "")
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
