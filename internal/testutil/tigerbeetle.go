package testutil

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TBInstance represents a running TigerBeetle test server.
type TBInstance struct {
	ClusterID string
	Addresses []string
	Stop      func()
}

// StartTigerBeetleSingleReplica launches a single-replica TigerBeetle instance.
func StartTigerBeetleSingleReplica(t *testing.T) *TBInstance {
	t.Helper()
	tbBin := lookupTBBinary(t)
	port := freePort(t)
	dir := t.TempDir()
	dataFile := filepath.Join(dir, "0_0.tigerbeetle")
	clusterID := "0"
	address := fmt.Sprintf("127.0.0.1:%d", port)

	formatCmd := exec.Command(tbBin, "format", "--cluster=0", "--replica=0", "--replica-count=1", "--development", dataFile)
	runCommand(t, formatCmd)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	startCmd := exec.Command(tbBin, "start", "--addresses="+address, "--development", dataFile)
	startCmd.Stdout = &stdout
	startCmd.Stderr = &stderr
	if err := startCmd.Start(); err != nil {
		t.Fatalf("tigerbeetle start failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	waitForPort(t, address, 5*time.Second)

	stop := func() {
		_ = startCmd.Process.Kill()
		_, _ = startCmd.Process.Wait()
	}
	t.Cleanup(stop)
	return &TBInstance{
		ClusterID: clusterID,
		Addresses: []string{address},
		Stop:      stop,
	}
}

func lookupTBBinary(t *testing.T) string {
	t.Helper()
	if env := os.Getenv("TB_BIN"); env != "" {
		return env
	}
	path, err := exec.LookPath("tigerbeetle")
	if err != nil {
		t.Skip("TB_BIN not set and tigerbeetle not found on PATH")
	}
	return path
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func waitForPort(t *testing.T, address string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("port %s did not become ready", address)
}

func runCommand(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("command failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
}
