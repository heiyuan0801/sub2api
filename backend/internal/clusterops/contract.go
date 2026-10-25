package clusterops

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

const contractKey = "cluster_security_contract_v1"

// The first explicitly configured node establishes the durable contract. It is
// never silently replaced during restart or failover. No raw secrets leave this
// process. Per-node addresses, roles and capacity intentionally differ.
func (m *Manager) contractFingerprint() (string, error) {
	if m.cfg.JWT.Secret == "" || m.cfg.Totp.EncryptionKey == "" {
		return "", errors.New("shared cluster credentials are required")
	}
	payload, err := json.Marshal(struct {
		TOTP          string
		Database      string
		RedisDB       int
		HA            bool
		StrictBilling bool
	}{m.cfg.Totp.EncryptionKey, m.cfg.Database.DBName, m.cfg.Redis.DB, m.cfg.Cluster.HAEnabled, m.cfg.Cluster.StrictBilling()})
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(m.cfg.JWT.Secret))
	_, _ = mac.Write(append([]byte("sub2api-cluster-contract-v1\x00"), payload...))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (m *Manager) verifyContract(ctx context.Context) bool {
	fingerprint, err := m.contractFingerprint()
	if err != nil {
		return false
	}
	if !m.contractRegistered.Load() {
		_, err = m.db.ExecContext(ctx, `INSERT INTO settings(key,value,updated_at) VALUES($1,$2,NOW()) ON CONFLICT(key) DO NOTHING`, contractKey, fingerprint)
		if err != nil {
			return false
		}
		m.contractRegistered.Store(true)
	}
	var expected string
	if err = m.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=$1`, contractKey).Scan(&expected); err != nil {
		return false
	}
	return hmac.Equal([]byte(expected), []byte(fingerprint))
}
