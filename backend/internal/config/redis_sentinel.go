package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func (r RedisConfig) ValidateSentinel() error {
	if r.SentinelMasterName == "" && len(r.SentinelAddrs) == 0 {
		return nil
	}
	if strings.TrimSpace(r.SentinelMasterName) == "" || len(r.SentinelAddrs) < 3 {
		return fmt.Errorf("redis sentinel requires a master name and at least three distinct sentinel addresses")
	}
	seen := map[string]bool{}
	for _, addr := range r.SentinelAddrs {
		host, port, err := net.SplitHostPort(addr)
		n, portErr := strconv.Atoi(port)
		if err != nil || host == "" || portErr != nil || n < 1 || n > 65535 || seen[addr] {
			return fmt.Errorf("invalid or duplicate Redis Sentinel address")
		}
		seen[addr] = true
	}
	return nil
}
