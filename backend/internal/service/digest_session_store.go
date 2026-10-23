package service

import (
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"strconv"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// digestSessionTTL 摘要会话默认 TTL
const digestSessionTTL = 5 * time.Minute

// sessionEntry flat cache 条目
type sessionEntry struct {
	uuid      string
	accountID int64
}

// DigestSessionStore 内存摘要会话存储（flat cache 实现）
// key: "{groupID}:{prefixHash}|{digestChain}" → *sessionEntry
type DigestSessionStore struct {
	cache  *gocache.Cache
	shared SharedSessionCache
}

type sharedDigestSession struct {
	UUID      string    `json:"uuid"`
	AccountID int64     `json:"account_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewDigestSessionStore 创建内存摘要会话存储
func NewDigestSessionStore() *DigestSessionStore {
	return &DigestSessionStore{
		cache: gocache.New(digestSessionTTL, time.Minute),
	}
}

func ProvideDigestSessionStore(cache GatewayCache, cfg *config.Config) *DigestSessionStore {
	s := NewDigestSessionStore()
	if cfg != nil && cfg.Cluster.Enabled {
		s.shared, _ = cache.(SharedSessionCache)
	}
	return s
}

// Save 保存摘要会话。oldDigestChain 为 Find 返回的 matchedChain，用于删旧 key。
func (s *DigestSessionStore) Save(groupID int64, prefixHash, digestChain, uuid string, accountID int64, oldDigestChain string) {
	if digestChain == "" {
		return
	}
	ns := buildNS(groupID, prefixHash)
	s.cache.Set(ns+digestChain, &sessionEntry{uuid: uuid, accountID: accountID}, gocache.DefaultExpiration)
	if s.shared != nil {
		value, _ := json.Marshal(sharedDigestSession{uuid, accountID, time.Now().Add(digestSessionTTL)})
		ctx, cancel := sessionCacheContext()
		sharedSessionWarning(s.shared.SetSessionValue(ctx, sessionMetadataKey("digest", ns, digestChain), string(value), digestSessionTTL))
		cancel()
	}
	if oldDigestChain != "" && oldDigestChain != digestChain {
		s.cache.Delete(ns + oldDigestChain)
	}
}

// Find 查找摘要会话，从完整 chain 逐段截断，返回最长匹配及对应 matchedChain。
func (s *DigestSessionStore) Find(groupID int64, prefixHash, digestChain string) (uuid string, accountID int64, matchedChain string, found bool) {
	if digestChain == "" {
		return "", 0, "", false
	}
	ns := buildNS(groupID, prefixHash)
	if s.shared != nil {
		if value, ok := s.cache.Get(ns + digestChain); ok {
			if entry, ok := value.(*sessionEntry); ok {
				return entry.uuid, entry.accountID, digestChain, true
			}
		}
		keys, chains := sessionDigestKeys(ns, digestChain)
		ctx, cancel := sessionCacheContext()
		values, err := s.shared.GetSessionValues(ctx, keys)
		cancel()
		sharedSessionWarning(err)
		for i, value := range values {
			var entry sharedDigestSession
			if json.Unmarshal([]byte(value), &entry) == nil && time.Now().Before(entry.ExpiresAt) && entry.UUID != "" {
				return entry.UUID, entry.AccountID, chains[i], true
			}
		}
	}
	chain := digestChain
	for {
		if val, ok := s.cache.Get(ns + chain); ok {
			if e, ok := val.(*sessionEntry); ok {
				return e.uuid, e.accountID, chain, true
			}
		}
		i := strings.LastIndex(chain, "-")
		if i < 0 {
			return "", 0, "", false
		}
		chain = chain[:i]
	}
}

// buildNS 构建 namespace 前缀
func buildNS(groupID int64, prefixHash string) string {
	return strconv.FormatInt(groupID, 10) + ":" + prefixHash + "|"
}
