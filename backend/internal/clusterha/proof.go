// Package clusterha implements the leased process boundary for HA nodes.
package clusterha

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

// Proof is private local IPC between the runner and its child, never an API token.
// A separate watchdog in the child stops background jobs if the runner dies.
type Proof struct {
	Token string `json:"token"`
	Role  string `json:"role"`
	Until int64  `json:"until_unix_nano"`
}

func CheckProof(path, token, role string, now time.Time) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return errors.New("HA supervisor proof unavailable")
	}
	var p Proof
	if json.Unmarshal(b, &p) != nil || token == "" || p.Token != token || p.Role != role || p.Until <= now.UnixNano() || p.Until > now.Add(10*time.Second).UnixNano() {
		return errors.New("HA supervisor proof expired or mismatched")
	}
	return nil
}

func WriteProof(path string, p Proof) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
