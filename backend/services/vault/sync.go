package vault

import (
	"context"
	"fmt"
	"log"
	"os"
)

// SyncToDevice restores the latest snapshot to rebuild a new device.
// The user logs into a fresh vulos install, enters their S3 credentials,
// and this function pulls the latest snapshot and restores their entire digital life.
func (v *Vault) SyncToDevice(ctx context.Context, targetDir string) error {
	if !v.status.Initialized {
		return fmt.Errorf("vault not initialized — configure S3 storage first")
	}

	// Get latest snapshot
	snaps, err := v.Snapshots(ctx)
	if err != nil {
		return fmt.Errorf("list snapshots: %w", err)
	}
	if len(snaps) == 0 {
		return fmt.Errorf("no snapshots found — nothing to restore")
	}

	// Use the most recent snapshot
	latest := snaps[len(snaps)-1]
	log.Printf("[vault/sync] restoring snapshot %s (%s) to %s", latest.ID, latest.Time, targetDir)

	// Ensure target directory exists
	os.MkdirAll(targetDir, 0755)

	// Restore
	if err := v.Restore(ctx, latest.ID, targetDir); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	log.Printf("[vault/sync] restore complete — device synced from snapshot %s", latest.ID)
	return nil
}

// SyncStatus returns information about the sync state between this device
// and the remote vault.
func (v *Vault) SyncStatus(ctx context.Context) map[string]any {
	status := map[string]any{
		"vault_initialized": v.status.Initialized,
		"last_backup":       v.status.LastBackup,
		"running":           v.status.Running,
	}

	if !v.status.Initialized {
		status["message"] = "Configure S3 storage to enable sync"
		return status
	}

	snaps, err := v.Snapshots(ctx)
	if err != nil {
		status["error"] = err.Error()
		return status
	}

	status["total_snapshots"] = len(snaps)
	if len(snaps) > 0 {
		latest := snaps[len(snaps)-1]
		status["latest_snapshot"] = latest.ID
		status["latest_time"] = latest.Time
		status["latest_hostname"] = latest.Hostname
	}

	// Check if current device matches latest snapshot hostname
	hostname, _ := os.Hostname()
	status["current_hostname"] = hostname

	if len(snaps) > 0 {
		otherDevices := make(map[string]bool)
		for _, s := range snaps {
			if s.Hostname != hostname {
				otherDevices[s.Hostname] = true
			}
		}
		devices := make([]string, 0, len(otherDevices))
		for d := range otherDevices {
			devices = append(devices, d)
		}
		status["other_devices"] = devices
	}

	return status
}
