package webbrowser

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Service manages a neko remote browser instance.
// Neko runs headless Chromium/Firefox with WebRTC streaming — 60fps, works everywhere.
// The vulos frontend iframes the neko web UI.
//
// Install: download neko binary or use the Docker image.
// Alpine: not in repos — download from GitHub releases.
type Service struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	port    int
	running bool
}

func New() *Service {
	return &Service{port: 3000}
}

// Start launches the neko browser server.
func (s *Service) Start(ctx context.Context, nekoPort int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	if nekoPort > 0 {
		s.port = nekoPort
	}

	// Try to find neko binary
	nekoBin := findBin("neko", "/usr/local/bin/neko", "/opt/neko/bin/neko")
	if nekoBin == "" {
		// Fallback: try running via Docker if available
		dockerBin := findBin("docker")
		if dockerBin != "" {
			return s.startViaDocker(ctx, dockerBin)
		}
		return fmt.Errorf("neko not found — install from https://github.com/m1k1o/neko or run via Docker")
	}

	s.cmd = exec.CommandContext(ctx, nekoBin)
	s.cmd.Env = append(os.Environ(),
		fmt.Sprintf("NEKO_BIND=0.0.0.0:%d", s.port),
		"NEKO_EPR=52000-52100",
		"NEKO_ICELITE=true",
		"NEKO_SCREEN=1280x720@30",
	)
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("start neko: %w", err)
	}

	s.running = true
	go func() {
		s.cmd.Wait()
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		log.Printf("[browser] neko exited")
	}()

	log.Printf("[browser] neko started on port %d", s.port)
	return nil
}

// startViaDocker launches neko as a Docker container.
func (s *Service) startViaDocker(ctx context.Context, dockerBin string) error {
	// Kill any existing neko container
	exec.CommandContext(ctx, dockerBin, "rm", "-f", "vulos-neko").Run()

	s.cmd = exec.CommandContext(ctx, dockerBin, "run",
		"--name", "vulos-neko",
		"--rm",
		"-p", fmt.Sprintf("%d:8080", s.port),
		"-p", "52000-52100:52000-52100/udp",
		"-e", "NEKO_SCREEN=1280x720@30",
		"-e", "NEKO_EPR=52000-52100",
		"-e", "NEKO_ICELITE=true",
		"-e", "NEKO_TCPMUX=52000",
		"-e", "NEKO_UDPMUX=52000",
		"-e", "NEKO_NAT1TO1="+getNAT1TO1(),
		"-e", "NEKO_PASSWORD=vulos",
		"-e", "NEKO_PASSWORD_ADMIN=vulos",
		"--shm-size=2g",
		"ghcr.io/m1k1o/neko/firefox:latest",
	)
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("start neko docker: %w", err)
	}

	s.running = true
	go func() {
		s.cmd.Wait()
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		log.Printf("[browser] neko container exited")
	}()

	log.Printf("[browser] neko started via Docker on port %d", s.port)
	return nil
}

// Stop kills the neko instance.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd != nil && s.cmd.Process != nil {
		syscall.Kill(-s.cmd.Process.Pid, syscall.SIGTERM)
		time.Sleep(time.Second)
		syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
	}
	// Also try Docker cleanup
	exec.Command("docker", "rm", "-f", "vulos-neko").Run()
	s.running = false
	s.cmd = nil
}

// Running returns whether neko is active.
func (s *Service) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Port returns the neko web UI port.
func (s *Service) Port() int {
	return s.port
}

// StopAll is an alias for Stop.
func (s *Service) StopAll() {
	s.Stop()
}

func getNAT1TO1() string {
	// Check env override first
	if ip := os.Getenv("NEKO_NAT1TO1"); ip != "" {
		return ip
	}
	// Try resolving host.docker.internal (Docker Desktop)
	if ips, err := net.LookupHost("host.docker.internal"); err == nil && len(ips) > 0 {
		return ips[0]
	}
	// Fallback to first non-loopback IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}

func findBin(names ...string) string {
	for _, name := range names {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	// Check if port is available as a hint
	return ""
}

// WaitReady waits for neko to be accessible.
func (s *Service) WaitReady(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", s.port), time.Second)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}
