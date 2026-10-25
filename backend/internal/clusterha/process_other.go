//go:build !unix

package clusterha

import "os/exec"

func configureChildProcess(c *exec.Cmd)         {}
func signalChildProcess(c *exec.Cmd, kill bool) { _ = c.Process.Kill() }
func notifySystemd(message string) error        { return nil }
