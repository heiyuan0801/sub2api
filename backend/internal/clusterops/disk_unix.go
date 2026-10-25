//go:build linux || darwin

package clusterops

import "golang.org/x/sys/unix"

func diskSpace() (uint64, uint64) {
	var s unix.Statfs_t
	if unix.Statfs(".", &s) != nil || s.Bsize <= 0 {
		return 0, 0
	}
	block := uint64(s.Bsize)
	return s.Blocks * block, s.Bavail * block
}
