package wifi

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// EthernetStatus is the state of wired network interfaces.
type EthernetStatus struct {
	Interface string `json:"interface"`
	Connected bool   `json:"connected"`
	IP        string `json:"ip,omitempty"`
	Speed     string `json:"speed,omitempty"` // e.g., "1000Mb/s"
	MAC       string `json:"mac,omitempty"`
}

// ListEthernet returns all wired (non-wireless, non-virtual) interfaces.
func ListEthernet(ctx context.Context) []EthernetStatus {
	out, err := output(ctx, "ip", "-o", "link", "show")
	if err != nil {
		return nil
	}

	var result []EthernetStatus
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		iface := strings.TrimSuffix(fields[1], ":")
		// Skip loopback, wireless, virtual, bridge, veth
		if iface == "lo" || strings.HasPrefix(iface, "wl") ||
			strings.HasPrefix(iface, "veth") || strings.HasPrefix(iface, "vh_") ||
			strings.HasPrefix(iface, "vn_") || strings.HasPrefix(iface, "br") ||
			strings.HasPrefix(iface, "docker") || strings.HasPrefix(iface, "vulos") {
			continue
		}
		// Only physical ethernet (usually eth* or en*)
		if !strings.HasPrefix(iface, "eth") && !strings.HasPrefix(iface, "en") {
			continue
		}

		es := EthernetStatus{Interface: iface}

		// Check link state
		if strings.Contains(line, "state UP") {
			es.Connected = true
		}

		// Get IP
		if ipOut, err := output(ctx, "ip", "-4", "addr", "show", iface); err == nil {
			for _, l := range strings.Split(string(ipOut), "\n") {
				l = strings.TrimSpace(l)
				if strings.HasPrefix(l, "inet ") {
					parts := strings.Fields(l)
					if len(parts) >= 2 {
						es.IP = strings.Split(parts[1], "/")[0]
					}
				}
			}
		}

		// Get MAC
		for i, f := range fields {
			if f == "link/ether" && i+1 < len(fields) {
				es.MAC = fields[i+1]
			}
		}

		// Get speed
		if speedOut, err := output(ctx, "ethtool", iface); err == nil {
			for _, l := range strings.Split(string(speedOut), "\n") {
				l = strings.TrimSpace(l)
				if strings.HasPrefix(l, "Speed:") {
					es.Speed = strings.TrimSpace(strings.TrimPrefix(l, "Speed:"))
				}
			}
		}

		result = append(result, es)
	}
	return result
}

// EnableDHCP starts dhcpcd on an ethernet interface.
func EnableDHCP(ctx context.Context, iface string) error {
	if err := run(ctx, "ip", "link", "set", iface, "up"); err != nil {
		return fmt.Errorf("bring up %s: %w", iface, err)
	}
	return run(ctx, "dhcpcd", iface)
}

// SetStaticIP configures a static IP on an ethernet interface.
func SetStaticIP(ctx context.Context, iface, ip, gateway string) error {
	if err := run(ctx, "ip", "link", "set", iface, "up"); err != nil {
		return err
	}
	if err := run(ctx, "ip", "addr", "flush", "dev", iface); err != nil {
		return err
	}
	if err := run(ctx, "ip", "addr", "add", ip, "dev", iface); err != nil {
		return err
	}
	if gateway != "" {
		return run(ctx, "ip", "route", "add", "default", "via", gateway, "dev", iface)
	}
	return nil
}

// DisableEthernet brings an interface down.
func DisableEthernet(ctx context.Context, iface string) error {
	exec.CommandContext(ctx, "dhcpcd", "-k", iface).Run() // release DHCP
	return run(ctx, "ip", "link", "set", iface, "down")
}
