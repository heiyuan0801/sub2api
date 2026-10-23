// cluster-config renders an ingress configuration without changing running services.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Wei-Shaw/sub2api/internal/clusterlb"
)

func main() {
	input := flag.String("config", "nodes.json", "server inventory JSON")
	flag.Parse()
	if err := run(*input); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	cfg, err := clusterlb.Read(f)
	if err != nil {
		return err
	}
	output, err := cfg.Render()
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, output)
	return err
}
