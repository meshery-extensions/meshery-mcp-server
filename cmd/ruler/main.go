// Copyright Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Metric represents a single architectural metric from the scorecard.
type Metric struct {
	ID         string  `yaml:"id"`
	Name       string  `yaml:"name"`
	Category   string  `yaml:"category"`
	Unit       string  `yaml:"unit"`
	Direction  string  `yaml:"direction"`
	Threshold  float64 `yaml:"threshold"`
	Standalone float64 `yaml:"standalone"`
	Adapter    float64 `yaml:"adapter"`
	Hybrid     float64 `yaml:"hybrid"`
	Notes      string  `yaml:"notes"`
}

// Scorecard is the top-level structure of benchmark/scorecard.yaml.
type Scorecard struct {
	Title   string   `yaml:"title"`
	Version string   `yaml:"version"`
	Spec    string   `yaml:"spec"`
	Metrics []Metric `yaml:"metrics"`
}

// jsonRPCRequest models a standard MCP tools/call payload for list_designs.
type jsonRPCRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      int            `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
}

// benchResult holds micro-benchmark results.
type benchResult struct {
	Iterations int
	Duration   time.Duration
	OpsPerSec  float64
	AvgLatency float64
	Payload    int
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)

	scorecardPath := filepath.Join("benchmark", "scorecard.yaml")
	data, err := os.ReadFile(scorecardPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read scorecard %s: %v\n", scorecardPath, err)
		os.Exit(1)
	}

	var sc Scorecard
	if err := yaml.Unmarshal(data, &sc); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse scorecard YAML: %v\n", err)
		os.Exit(1)
	}

	bench := runMicroBenchmark()

	renderHeader(sc, bench)
	renderScorecard(sc.Metrics)
	renderSummary(sc.Metrics)
	renderFooter(bench)
}

func runMicroBenchmark() benchResult {
	payload := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "list_designs",
			"arguments": map[string]any{
				"page":      1,
				"page_size": 10,
				"search":    "",
			},
		},
	}

	const iterations = 50000

	// Warm up JSON encoder to avoid cold-start noise.
	for i := 0; i < 1000; i++ {
		_, _ = json.Marshal(payload)
	}

	start := time.Now()
	var payloadSize int
	for i := 0; i < iterations; i++ {
		// Vary ID to prevent over-optimization.
		payload.ID = i
		b, err := json.Marshal(payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal failed at iteration %d: %v\n", i, err)
			os.Exit(1)
		}
		// Also measure unmarshal to cover full round-trip.
		var decoded jsonRPCRequest
		if err := json.Unmarshal(b, &decoded); err != nil {
			fmt.Fprintf(os.Stderr, "unmarshal failed at iteration %d: %v\n", i, err)
			os.Exit(1)
		}
		if i == 0 {
			payloadSize = len(b)
		}
	}
	elapsed := time.Since(start)

	opsPerSec := float64(iterations) / elapsed.Seconds()
	avgLatencyUs := float64(elapsed.Nanoseconds()) / float64(iterations) / 1000.0

	return benchResult{
		Iterations: iterations,
		Duration:   elapsed,
		OpsPerSec:  opsPerSec,
		AvgLatency: avgLatencyUs,
		Payload:    payloadSize,
	}
}

func renderHeader(sc Scorecard, b benchResult) {
	fmt.Printf("\n")
	fmt.Printf("  %s\n", sc.Title)
	fmt.Printf("  Version: %s  |  Spec: %s\n", sc.Version, sc.Spec)
	fmt.Printf("  ---------------------------------------------------------------------------\n")
	fmt.Printf("  Micro-benchmark: JSON-RPC tools/call (list_designs) round-trip\n")
	fmt.Printf("  Iterations: %d  |  Payload: %d bytes  |  Duration: %s\n", b.Iterations, b.Payload, b.Duration.Round(time.Millisecond))
	fmt.Printf("  Throughput: %.0f ops/sec  |  Avg latency: %.2f us/op\n", b.OpsPerSec, b.AvgLatency)
	fmt.Printf("  ---------------------------------------------------------------------------\n")
}

func renderScorecard(metrics []Metric) {
	fmt.Printf("\n")
	fmt.Printf("  %-6s | %-32s | %-8s | %-10s | %-10s | %-10s | %-12s\n",
		"ID", "Metric", "Unit", "Standalone", "Adapter", "Hybrid", "Superiority")
	fmt.Printf("  %-6s-+-%-32s-+-%-8s-+-%-10s-+-%-10s-+-%-10s-+-%-12s\n",
		"------", "--------------------------------", "--------", "----------", "----------", "----------", "------------")

	for _, m := range metrics {
		sup := superiority(m)
		supStr := fmt.Sprintf("%.1f%%", sup)
		// Mark threshold breach with asterisk.
		standaloneStr := fmt.Sprintf("%.1f", m.Standalone)
		adapterStr := fmt.Sprintf("%.1f", m.Adapter)
		hybridStr := fmt.Sprintf("%.1f", m.Hybrid)
		if isBreach(m, m.Standalone) {
			standaloneStr += "*"
		}
		if isBreach(m, m.Adapter) {
			adapterStr += "*"
		}
		if isBreach(m, m.Hybrid) {
			hybridStr += "*"
		}
		fmt.Printf("  %-6s | %-32s | %-8s | %-10s | %-10s | %-10s | %-12s\n",
			m.ID, truncate(m.Name, 32), m.Unit, standaloneStr, adapterStr, hybridStr, supStr)
	}
	fmt.Printf("\n  * threshold breach  |  Superiority = standalone advantage over adapter\n")
}

func renderSummary(metrics []Metric) {
	standaloneWins := 0
	adapterWins := 0
	hybridWins := 0
	thresholdPassStandalone := 0
	thresholdPassAdapter := 0
	thresholdPassHybrid := 0

	for _, m := range metrics {
		// Determine winner for this metric.
		best := bestValue(m)
		if m.Standalone == best {
			standaloneWins++
		}
		if m.Adapter == best {
			adapterWins++
		}
		if m.Hybrid == best {
			hybridWins++
		}
		if !isBreach(m, m.Standalone) {
			thresholdPassStandalone++
		}
		if !isBreach(m, m.Adapter) {
			thresholdPassAdapter++
		}
		if !isBreach(m, m.Hybrid) {
			thresholdPassHybrid++
		}
	}

	fmt.Printf("\n")
	fmt.Printf("  Summary\n")
	fmt.Printf("  ---------------------------------------------------------------------------\n")
	fmt.Printf("  %-32s : %d / %d\n", "Standalone wins", standaloneWins, len(metrics))
	fmt.Printf("  %-32s : %d / %d\n", "Adapter wins", adapterWins, len(metrics))
	fmt.Printf("  %-32s : %d / %d\n", "Hybrid wins", hybridWins, len(metrics))
	fmt.Printf("  ---------------------------------------------------------------------------\n")
	fmt.Printf("  %-32s : %d / %d\n", "Standalone threshold pass", thresholdPassStandalone, len(metrics))
	fmt.Printf("  %-32s : %d / %d\n", "Adapter threshold pass", thresholdPassAdapter, len(metrics))
	fmt.Printf("  %-32s : %d / %d\n", "Hybrid threshold pass", thresholdPassHybrid, len(metrics))
	fmt.Printf("  ---------------------------------------------------------------------------\n")

	// Overall superiority average for standalone vs adapter.
	var totalSup float64
	for _, m := range metrics {
		totalSup += superiority(m)
	}
	avgSup := totalSup / float64(len(metrics))
	fmt.Printf("  Average standalone superiority over adapter: %.1f%%\n", avgSup)
}

func renderFooter(b benchResult) {
	fmt.Printf("\n")
	fmt.Printf("  Notes: Benchmark is in-memory JSON-RPC serialization round-trip for a\n")
	fmt.Printf("  standard list_designs payload. It measures codec overhead, not network.\n")
	fmt.Printf("  Scorecard values are rated baselines on Go 1.26.0, linux/amd64,\n")
	fmt.Printf("  CGO_ENABLED=0. See benchmark/scorecard.yaml for per-metric notes.\n")
	fmt.Printf("\n")
}

// superiority returns standalone advantage over adapter as a percentage.
// For lower_is_better: (adapter - standalone) / adapter * 100
// For higher_is_better: (standalone - adapter) / standalone * 100
func superiority(m Metric) float64 {
	if m.Direction == "lower_is_better" {
		if m.Adapter == 0 {
			return 0
		}
		return (m.Adapter - m.Standalone) / m.Adapter * 100
	}
	if m.Adapter == 0 && m.Standalone == 0 {
		return 0
	}
	if m.Standalone == 0 {
		return 0
	}
	return (m.Standalone - m.Adapter) / m.Standalone * 100
}

// isBreach reports whether a value breaches its threshold.
func isBreach(m Metric, value float64) bool {
	if m.Direction == "lower_is_better" {
		return value > m.Threshold
	}
	return value < m.Threshold
}

// bestValue returns the best value among the three for a metric.
func bestValue(m Metric) float64 {
	if m.Direction == "lower_is_better" {
		best := m.Standalone
		if m.Adapter < best {
			best = m.Adapter
		}
		if m.Hybrid < best {
			best = m.Hybrid
		}
		return best
	}
	best := m.Standalone
	if m.Adapter > best {
		best = m.Adapter
	}
	if m.Hybrid > best {
		best = m.Hybrid
	}
	return best
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
