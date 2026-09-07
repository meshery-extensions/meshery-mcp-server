package mesherytest_test

import (
	"strings"
	"testing"

	"github.com/meshery-extensions/meshery-mcp-server/internal/mesherytest"
)

// The two assertions here state that something did not happen: a route was not
// called, a parameter was not sent. Both are exported, and both need a fixture
// where the thing did happen as well as one where it did not, since an
// assertion that never fires is indistinguishable from one with an empty body.

// TestAssertNotCalledSeesBothAnswers covers the negative half of a read-only
// guarantee. A server that never touches a mutating route and one that touches
// it must not look the same to this assertion.
func TestAssertNotCalledSeesBothAnswers(t *testing.T) {
	const mutating = "/api/pattern/deploy"

	t.Run("nothing called it", func(t *testing.T) {
		s := mesherytest.New(t)
		authedGet(t, s, "/api/pattern", "")
		if failed, msg := (&recorder{}).run(func(tt mesherytest.T) {
			s.AssertNotCalled(tt, mutating)
		}); failed {
			t.Errorf("no request went to %s, so the assertion should hold: %s", mutating, msg)
		}
	})

	t.Run("something called it", func(t *testing.T) {
		s := mesherytest.New(t)
		// The route need not exist for the fake to record the request: what is
		// being asserted on is what the client sent.
		authedGet(t, s, mutating, "")
		failed, msg := (&recorder{}).run(func(tt mesherytest.T) {
			s.AssertNotCalled(tt, mutating)
		})
		if !failed {
			t.Fatalf("a request went to %s and the assertion did not fire", mutating)
		}
		if !strings.Contains(msg, mutating) {
			t.Errorf("the message should name the route, got %q", msg)
		}
	})
}

// TestAssertNoQuerySeesBothAnswers covers the parameter a read-only posture
// depends on never sending. Asking MeshSync for spec or status is what puts a
// Secret payload on the wire, so an assertion that cannot tell the two cases
// apart is worse than none.
func TestAssertNoQuerySeesBothAnswers(t *testing.T) {
	t.Run("parameter absent", func(t *testing.T) {
		s := mesherytest.New(t)
		authedGet(t, s, resourcesPath, `clusterIds=["`+s.Data().ClusterID()+`"]`)
		if failed, msg := (&recorder{}).run(func(tt mesherytest.T) {
			s.AssertNoQuery(tt, resourcesPath, "spec")
		}); failed {
			t.Errorf("spec was never sent, so the assertion should hold: %s", msg)
		}
	})

	t.Run("parameter present", func(t *testing.T) {
		s := mesherytest.New(t)
		authedGet(t, s, resourcesPath, `clusterIds=["`+s.Data().ClusterID()+`"]&spec=true`)
		failed, msg := (&recorder{}).run(func(tt mesherytest.T) {
			s.AssertNoQuery(tt, resourcesPath, "spec")
		})
		if !failed {
			t.Fatal("spec was sent and the assertion did not fire")
		}
		if !strings.Contains(msg, "spec") {
			t.Errorf("the message should name the parameter, got %q", msg)
		}
	})

	t.Run("present but empty still counts as sent", func(t *testing.T) {
		s := mesherytest.New(t)
		// A bare key with no value is still a key on the wire, and Meshery
		// parses it as present. An assertion reading the value rather than
		// testing for the key would miss this.
		authedGet(t, s, resourcesPath, `clusterIds=["`+s.Data().ClusterID()+`"]&status=`)
		if failed, _ := (&recorder{}).run(func(tt mesherytest.T) {
			s.AssertNoQuery(tt, resourcesPath, "status")
		}); !failed {
			t.Error("status was sent with an empty value and the assertion did not fire")
		}
	})
}
