package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
)

// SharedSessionCache stores only routing metadata and opaque continuation state,
// never complete prompts or live connection handles. Implementations hash keys.
type SharedSessionCache interface {
	GetSessionValues(context.Context, []string) ([]string, error)
	SetSessionValue(context.Context, string, string, time.Duration) error
	DeleteSessionValue(context.Context, string) error
}

type sharedResponseRouteWriter interface {
	SetResponseRoute(context.Context, int64, string, string, int64, string, string, time.Duration) error
}

var sessionCacheFailures atomic.Uint64
var sessionCacheLastWarning atomic.Int64

func sharedSessionWarning(err error) {
	if err == nil {
		return
	}
	sessionCacheFailures.Add(1)
	now := time.Now().Unix()
	last := sessionCacheLastWarning.Load()
	if now-last >= 30 && sessionCacheLastWarning.CompareAndSwap(last, now) {
		// Error text can contain infrastructure addresses; keep routing material out.
		slog.Warn("cluster session cache unavailable", "failures", sessionCacheFailures.Load())
	}
}

func sessionCacheContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Second)
}

func sessionMetadataKey(kind, namespace, value string) string {
	sum := sha256.Sum256([]byte(namespace + "\x00" + value))
	return kind + ":" + hex.EncodeToString(sum[:])
}

// Bound both command size and lookup cost for untrusted, long conversation chains.
// One MGET per lookup avoids a network round trip per historical message.
func sessionDigestKeys(namespace, chain string) (keys, chains []string) {
	for chain != "" && len(keys) < 256 {
		keys = append(keys, sessionMetadataKey("digest", namespace, chain))
		chains = append(chains, chain)
		i := strings.LastIndex(chain, "-")
		if i < 0 {
			break
		}
		chain = chain[:i]
	}
	return
}
