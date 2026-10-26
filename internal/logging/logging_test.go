package logging

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
)

const app = "test-app"

// lines splits log output into non-empty lines.
func lines(buf *bytes.Buffer) []string {
	var out []string
	for _, l := range strings.Split(buf.String(), "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

func TestNew_JSONLinesHaveOneShape(t *testing.T) {
	var buf bytes.Buffer
	mk, sdk, err := New(app, config.LogFormatJSON, config.LogLevelInfo, &buf)
	if err != nil {
		t.Fatal(err)
	}
	mk.Info("from meshkit")
	sdk.Info("from sdk")
	mk.Warnf("warn from meshkit")
	sdk.Warn("warn from sdk")

	got := lines(&buf)
	if len(got) != 4 {
		t.Fatalf("got %d lines, want 4:\n%s", len(got), buf.String())
	}
	var entries []map[string]any
	for _, l := range got {
		var e map[string]any
		if err := json.Unmarshal([]byte(l), &e); err != nil {
			t.Fatalf("line is not JSON: %q", l)
		}
		entries = append(entries, e)
	}

	wantKeys := []string{"app", "level", "msg", "time"}
	for i, e := range entries {
		if keys := slices.Sorted(maps.Keys(e)); !slices.Equal(keys, wantKeys) {
			t.Errorf("line %d keys = %v, want %v", i, keys, wantKeys)
		}
		if e["app"] != app {
			t.Errorf("line %d app = %v, want %q", i, e["app"], app)
		}
	}
	for i, want := range []string{"info", "info", "warning", "warning"} {
		if entries[i]["level"] != want {
			t.Errorf("line %d level = %v, want %q", i, entries[i]["level"], want)
		}
	}
	if entries[0]["time"].(string) == "" || len(entries[0]["time"].(string)) != len(entries[1]["time"].(string)) {
		t.Errorf("time formats differ: %q vs %q", entries[0]["time"], entries[1]["time"])
	}
}

func TestNew_TextLinesUseSameFields(t *testing.T) {
	var buf bytes.Buffer
	mk, sdk, err := New(app, config.LogFormatText, config.LogLevelInfo, &buf)
	if err != nil {
		t.Fatal(err)
	}
	mk.Info("from meshkit")
	sdk.Info("from sdk")

	for _, l := range lines(&buf) {
		for _, want := range []string{"level=info", "app=" + app, "msg="} {
			if !strings.Contains(l, want) {
				t.Errorf("line %q does not contain %q", l, want)
			}
		}
		if strings.HasPrefix(l, "{") {
			t.Errorf("text line looks like JSON: %q", l)
		}
	}
}

func TestNew_LevelAppliesToBoth(t *testing.T) {
	var buf bytes.Buffer
	mk, sdk, err := New(app, config.LogFormatText, config.LogLevelWarn, &buf)
	if err != nil {
		t.Fatal(err)
	}
	mk.Info("hidden")
	sdk.Info("hidden")
	mk.Warnf("shown")
	sdk.Warn("shown")

	out := buf.String()
	if strings.Contains(out, "hidden") {
		t.Errorf("info lines were written at warn level:\n%s", out)
	}
	if n := strings.Count(out, "shown"); n != 2 {
		t.Errorf("got %d warn lines, want 2:\n%s", n, out)
	}
}
