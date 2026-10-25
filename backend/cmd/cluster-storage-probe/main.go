// cluster-storage-probe writes only a unique 1 KiB operational canary on explicit request.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/clusterprobe"
	"os"
	"path/filepath"
	"time"
)

func run() error {
	action := flag.String("action", "", "write, read or cleanup")
	node := flag.String("node", "", "local cluster node ID")
	challenge := flag.String("challenge", "", "fresh shared UUID")
	input := flag.String("receipt", "", "peer write receipt for read; own write receipt for cleanup")
	output := flag.String("output", "", "new receipt file")
	local := flag.Bool("local-lab", false, "allow loopback HTTP and same-host canaries")
	write := flag.Bool("write-probe", false, "authorize canary creation or cleanup")
	flag.Parse()
	if (*action == "write" || *action == "read") && *output == "" {
		return fmt.Errorf("receipt output is required")
	}
	if (*action == "write" || *action == "cleanup") && !*write {
		return fmt.Errorf("canary writes require --write-probe")
	}
	c, e := clusterprobe.New(os.Getenv("SUB2API_PROBE_S3_ENDPOINT"), os.Getenv("SUB2API_PROBE_S3_REGION"), os.Getenv("SUB2API_PROBE_S3_BUCKET"), os.Getenv("SUB2API_PROBE_S3_ACCESS_KEY"), os.Getenv("SUB2API_PROBE_S3_SECRET_KEY"), *local)
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var in, out clusterprobe.Receipt
	if *input != "" {
		b, err := os.ReadFile(*input)
		if err != nil || len(b) > 4096 {
			return clusterprobe.ErrProbe
		}
		if json.Unmarshal(b, &in) != nil {
			return clusterprobe.ErrProbe
		}
	}
	switch *action {
	case "write":
		out, e = c.Write(ctx, *challenge, *node)
	case "read":
		out, e = c.Read(ctx, *node, in)
	case "cleanup":
		return c.Cleanup(ctx, in)
	default:
		return fmt.Errorf("choose write, read or cleanup")
	}
	if e != nil {
		return e
	}
	if *output == "" {
		return fmt.Errorf("receipt output is required")
	}
	b, e := json.MarshalIndent(out, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(*output), ".storage-probe-")
	if e != nil {
		return e
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, e = f.Write(b); e != nil {
		_ = f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), *output)
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	fmt.Println("Storage probe completed")
}
