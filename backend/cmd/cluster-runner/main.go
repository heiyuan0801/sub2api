package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Wei-Shaw/sub2api/internal/clusterha"
)

func main() {
	file := flag.String("config", "", "runner configuration JSON")
	flag.Parse()
	cfg, err := clusterha.ReadConfig(*file)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := clusterha.Run(ctx, cfg); err != nil {
		log.Fatal(err)
	}
}
