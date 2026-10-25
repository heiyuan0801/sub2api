package clusterops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrivateIPCRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proof.control")
	require.NoError(t, writePrivateJSON(path, map[string]bool{"maintenance": true}))
	b, err := readPrivateFile(path)
	require.NoError(t, err)
	require.JSONEq(t, `{"maintenance":true}`, string(b))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	require.NoError(t, writePrivateJSON(path, map[string]bool{"maintenance": false}))
	b, err = readPrivateFile(path)
	require.NoError(t, err)
	require.JSONEq(t, `{"maintenance":false}`, string(b))
	_, err = os.Stat(path + ".tmp")
	require.True(t, os.IsNotExist(err))
}

func TestPrivateIPCRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.json")
	require.NoError(t, os.WriteFile(outside, []byte(`{"keep":true}`), 0600))
	ack := filepath.Join(root, "proof.proxy-ack")
	require.NoError(t, os.Symlink(outside, ack))
	_, err := readPrivateFile(ack)
	require.Error(t, err)
	control := filepath.Join(root, "proof.control")
	require.NoError(t, os.Symlink(outside, control+".tmp"))
	require.Error(t, writePrivateJSON(control, map[string]bool{"keep": false}))
	b, err := os.ReadFile(outside)
	require.NoError(t, err)
	require.JSONEq(t, `{"keep":true}`, string(b))
}
