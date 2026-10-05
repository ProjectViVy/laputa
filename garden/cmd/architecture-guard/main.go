package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ProjectViVy/laputa/garden/internal/architectureguard"
)

func main() {
	root := flag.String("root", ".", "repository root to scan")
	mode := flag.String("mode", string(architectureguard.ModeReport), "report or enforce")
	jsonOutput := flag.Bool("json", false, "emit a JSON finding list")
	flag.Parse()

	selected := architectureguard.Mode(*mode)
	if selected != architectureguard.ModeReport && selected != architectureguard.ModeEnforce {
		fmt.Fprintf(os.Stderr, "invalid mode %q: use report or enforce\n", *mode)
		os.Exit(2)
	}
	findings, err := architectureguard.Scan(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "architecture guard: %v\n", err)
		os.Exit(2)
	}
	if *jsonOutput {
		if findings == nil {
			findings = []architectureguard.Finding{}
		}
		if err := json.NewEncoder(os.Stdout).Encode(findings); err != nil {
			fmt.Fprintf(os.Stderr, "architecture guard output: %v\n", err)
			os.Exit(2)
		}
	} else {
		for _, finding := range findings {
			fmt.Printf("%s:%d [%s] %s\n", finding.Path, finding.Line, finding.Rule, finding.Match)
		}
		fmt.Printf("architecture guard: %d violation(s)\n", len(findings))
	}
	os.Exit(architectureguard.ExitCode(selected, len(findings)))
}
