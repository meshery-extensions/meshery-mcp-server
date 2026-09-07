package mcpera

import (
	"strings"
	"testing"
)

// TestClassifyReadsBothModernSignalsIndependently is the disagreeing fixture for
// the era verdict.
//
// A server counts as modern either because it answered server/discover or
// because it served a modern call and returned a modern-shaped result. Both are
// true of a normal dual-era server, so fixtures built from real servers make the
// two agree, and either disjunct could be deleted with the suite still green.
// The revision makes server/discover optional, so the disagreeing cases are not
// hypothetical: a server may implement one and not the other.
func TestClassifyReadsBothModernSignalsIndependently(t *testing.T) {
	for _, tc := range []struct {
		name              string
		answersDiscover   bool
		servesModernCall  bool
		modernResult      bool
		answersInitialize bool
		want              Era
	}{
		// Only the discover half is true. server/discover is optional, so a
		// server may answer it without serving the call this probe made.
		{"discover alone", true, false, false, false, Modern},
		// Only the served-call half is true, which is what a server that
		// declined to implement the optional probe looks like.
		{"served call alone", false, true, true, false, Modern},
		// The served call was answered but in the legacy shape, which is not a
		// modern signal on its own.
		{"served call, legacy shape", false, true, false, false, Unknown},
		// Each modern signal alone, alongside the legacy opening, is dual.
		{"discover alone plus initialize", true, false, false, true, Dual},
		{"served call alone plus initialize", false, true, true, true, Dual},
		{"initialize only", false, false, false, true, Legacy},
		{"nothing", false, false, false, false, Unknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &Report{
				AnswersDiscover:      tc.answersDiscover,
				ServesModernCall:     tc.servesModernCall,
				ModernResultIsModern: tc.modernResult,
				AnswersInitialize:    tc.answersInitialize,
			}
			r.classify()
			if r.Era != tc.want {
				t.Errorf("Era = %v, want %v (discover=%v servedCall=%v modernShape=%v initialize=%v)",
					r.Era, tc.want, tc.answersDiscover, tc.servesModernCall, tc.modernResult, tc.answersInitialize)
			}
		})
	}
}

// TestSilentDowngradeNeedsBothHalves pins the other compound verdict. It fires
// only when a modern call was served and the result came back in the legacy
// shape, so neither half alone is enough.
func TestSilentDowngradeNeedsBothHalves(t *testing.T) {
	for _, tc := range []struct {
		name         string
		served       bool
		modernResult bool
		want         bool
	}{
		{"served with a legacy-shaped result", true, false, true},
		{"served with a modern result", true, true, false},
		{"never served the call", false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &Report{ServesModernCall: tc.served, ModernResultIsModern: tc.modernResult}
			r.classify()
			if r.SilentDowngrade != tc.want {
				t.Errorf("SilentDowngrade = %v, want %v", r.SilentDowngrade, tc.want)
			}
		})
	}
}

// TestDiscoverNeedsARealDiscoverResult covers the gap between answering a
// method and implementing it.
//
// DiscoverResult is what the revision replaced the initialize handshake with,
// and schema/2026-07-28/schema.json marks supportedVersions, capabilities,
// resultType, cacheScope and ttlMs required. A server returning anything less
// has not told a client what to negotiate, and counting it puts that server in
// the dual-era row, the one a client trusts most.
func TestDiscoverNeedsARealDiscoverResult(t *testing.T) {
	const full = `{"supportedVersions":["2026-07-28"],"capabilities":{},` +
		`"resultType":"complete","cacheScope":"private","ttlMs":60000}`

	for _, tc := range []struct {
		name   string
		result string
		want   bool
	}{
		{"a real discover answer", full, true},
		// Every required field present, but the server does not list the version
		// the probe declared, so there is nothing agreed to speak.
		{"supports another version only", `{"supportedVersions":["2025-11-25"],"capabilities":{},` +
			`"resultType":"complete","cacheScope":"private","ttlMs":60000}`, false},
		{"no supportedVersions", `{"capabilities":{},"resultType":"complete",` +
			`"cacheScope":"private","ttlMs":60000}`, false},
		{"no capabilities", `{"supportedVersions":["2026-07-28"],"resultType":"complete",` +
			`"cacheScope":"private","ttlMs":60000}`, false},
		{"no ttlMs", `{"supportedVersions":["2026-07-28"],"capabilities":{},` +
			`"resultType":"complete","cacheScope":"private"}`, false},
		// ttlMs of zero is a legitimate hint, so its presence is what counts
		// rather than its value.
		{"ttlMs of zero still counts", `{"supportedVersions":["2026-07-28"],"capabilities":{},` +
			`"resultType":"complete","cacheScope":"private","ttlMs":0}`, true},
		{"empty object", `{}`, false},
		{"null", `null`, false},
		{"absent", ``, false},
		{"not an object", `"ok"`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := describesAServer([]byte(tc.result)); got != tc.want {
				t.Errorf("describesAServer(%s) = %v, want %v", tc.result, got, tc.want)
			}
		})
	}
}

// TestSilenceIsNotADeterministicFailure separates the two ways server/discover
// can fail to succeed. A server that replies with an error has told a client
// something it can code against. A server that hung or died has told it nothing,
// and crediting that with a deterministic refusal reports a fact about the
// method on the strength of a fact about the process.
func TestSilenceIsNotADeterministicFailure(t *testing.T) {
	for _, tc := range []struct {
		name       string
		answered   bool
		errText    string
		wantInNote string
	}{
		{"refused with an error", true, "-32601: method not found", "fails deterministically"},
		{"answered but described nothing", true, "answered without describing a server", "fails deterministically"},
		{"hung", false, string(silentTimeout), "says nothing about whether the method is implemented"},
		{"died", false, string(silentEOF), "says nothing about whether the method is implemented"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &Report{DiscoverError: tc.errText, DiscoverAnswered: tc.answered}
			r.classify()
			joined := strings.Join(r.Notes, " ")
			if !strings.Contains(joined, tc.wantInNote) {
				t.Errorf("notes = %v, want one containing %q", r.Notes, tc.wantInNote)
			}
			if !tc.answered && strings.Contains(joined, "fails deterministically") {
				t.Errorf("silence was reported as a deterministic failure: %v", r.Notes)
			}
		})
	}
}

// TestDiscoverSuccessAddsNoFailureNote is the control: a server that answered
// gets neither note.
func TestDiscoverSuccessAddsNoFailureNote(t *testing.T) {
	r := &Report{AnswersDiscover: true, AnswersInitialize: true}
	r.classify()
	joined := strings.Join(r.Notes, " ")
	if strings.Contains(joined, "deterministically") || strings.Contains(joined, "no answer") {
		t.Errorf("a working server should carry no discover failure note: %v", r.Notes)
	}
}
