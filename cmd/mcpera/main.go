// Command mcpera reports which MCP protocol era a server serves.
//
//	mcpera ./my-mcp-server
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/meshery-extensions/meshery-mcp-server/internal/mcpera"
)

func main() {
	timeout := flag.Duration("timeout", 5*time.Second, "per-probe timeout")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: mcpera [flags] <server> [args...]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	rep, err := mcpera.Probe(context.Background(), *timeout, flag.Arg(0), flag.Args()[1:]...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mcpera:", err)
		os.Exit(1)
	}

	fmt.Printf("era: %s\n\n", rep.Era)
	fmt.Printf("  initialize          %s\n", yesNo(rep.AnswersInitialize, rep.NegotiatedVersion))
	fmt.Printf("  server/discover     %s\n", yesNo(rep.AnswersDiscover, rep.DiscoverError))
	fmt.Printf("  modern request      %s\n", yesNo(rep.ServesModernCall, ""))
	fmt.Printf("  modern result shape %s\n", yesNo(rep.ModernResultIsModern, ""))
	fmt.Println()
	for _, n := range rep.Notes {
		fmt.Printf("  %s\n", n)
	}

	if rep.SilentDowngrade {
		os.Exit(1)
	}
}

func yesNo(ok bool, detail string) string {
	s := "no"
	if ok {
		s = "yes"
	}
	if detail != "" {
		s += "  (" + detail + ")"
	}
	return s
}
