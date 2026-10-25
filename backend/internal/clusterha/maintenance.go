package clusterha

import (
	"encoding/json"
	"os"
)

// MaintenanceControl is written only by the local application into its private
// proof directory. Maintenance survives child restarts until explicitly resumed.
type MaintenanceControl struct {
	NodeID      string `json:"node_id"`
	Maintenance bool   `json:"maintenance"`
	Drained     bool   `json:"drained"`
}

func ReadMaintenance(path, node string) (MaintenanceControl, error) {
	var c MaintenanceControl
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err = json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if c.NodeID != node {
		return c, os.ErrPermission
	}
	return c, nil
}
