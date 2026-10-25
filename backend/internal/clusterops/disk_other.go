//go:build !linux && !darwin

package clusterops

func diskSpace() (uint64, uint64) { return 0, 0 }
