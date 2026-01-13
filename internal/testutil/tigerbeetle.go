package testutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

var (
	tbBaseOnce sync.Once
	tbBasePath string
	tbBaseErr  error
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

	baseFile := baseTigerBeetleDataFile(t, tbBin)
	copyTigerBeetleDataFile(t, baseFile, dataFile)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	startCmd := exec.Command(tbBin, "start", "--addresses="+address, dataFile)
	startCmd.Stdout = &stdout
	startCmd.Stderr = &stderr
	if err := startCmd.Start(); err != nil {
		t.Fatalf("tigerbeetle start failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	waitForPort(t, address, 20*time.Second)

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

// baseTigerBeetleDataFile returns a cached formatted TigerBeetle data file path.
func baseTigerBeetleDataFile(t *testing.T, tbBin string) string {
	t.Helper()
	tbBaseOnce.Do(func() {
		candidates := tigerBeetleCacheDirs()
		basePath, cacheErr := findExistingBaseFile(candidates)
		if cacheErr != nil {
			tbBaseErr = cacheErr
			return
		}
		if basePath != "" {
			tbBasePath = basePath
			return
		}

		cacheDir, err := firstWritableDir(candidates)
		if err != nil {
			tbBaseErr = fmt.Errorf("create tigerbeetle cache dir: %w", err)
			return
		}
		basePath = filepath.Join(cacheDir, "tigerbeetle-base-0_0.tigerbeetle")

		lockPath := basePath + ".lock"
		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			if !errors.Is(err, os.ErrExist) {
				tbBaseErr = fmt.Errorf("acquire tigerbeetle base lock: %w", err)
				return
			}
			if err := waitForFile(basePath, 2*time.Minute); err != nil {
				tbBaseErr = err
				return
			}
			tbBasePath = basePath
			return
		}
		defer func() {
			_ = lockFile.Close()
			_ = os.Remove(lockPath)
		}()

		tmpPath := filepath.Join(cacheDir, fmt.Sprintf("tigerbeetle-base-%d.tmp", time.Now().UnixNano()))
		formatCmd := exec.Command(tbBin, "format", "--cluster=0", "--replica=0", "--replica-count=1", tmpPath)
		runCommand(t, formatCmd)
		if err := os.Rename(tmpPath, basePath); err != nil {
			_ = os.Remove(tmpPath)
			tbBaseErr = fmt.Errorf("commit tigerbeetle base file: %w", err)
			return
		}
		tbBasePath = basePath
	})
	if tbBaseErr != nil {
		t.Fatalf("prepare tigerbeetle base file: %v", tbBaseErr)
	}
	return tbBasePath
}

// tigerBeetleCacheDirs returns candidate directories for cached TB data files.
func tigerBeetleCacheDirs() []string {
	seen := map[string]struct{}{}
	var dirs []string
	if cacheRoot := os.Getenv("XDG_CACHE_HOME"); cacheRoot != "" {
		appendUniqueDir(&dirs, seen, filepath.Join(cacheRoot, "cogni", "tigerbeetle"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		appendUniqueDir(&dirs, seen, filepath.Join(home, ".cache", "cogni", "tigerbeetle"))
	}
	if cacheDir, err := os.UserCacheDir(); err == nil && cacheDir != "" {
		appendUniqueDir(&dirs, seen, filepath.Join(cacheDir, "cogni", "tigerbeetle"))
	}
	appendUniqueDir(&dirs, seen, filepath.Join(os.TempDir(), "cogni", "tigerbeetle"))
	return dirs
}

// findExistingBaseFile checks candidate directories for an existing base file.
func findExistingBaseFile(candidates []string) (string, error) {
	for _, candidate := range candidates {
		basePath := filepath.Join(candidate, "tigerbeetle-base-0_0.tigerbeetle")
		if _, err := os.Stat(basePath); err == nil {
			return basePath, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("stat tigerbeetle base file: %w", err)
		}
	}
	return "", nil
}

// firstWritableDir returns the first candidate directory that can be created.
func firstWritableDir(candidates []string) (string, error) {
	var lastErr error
	for _, candidate := range candidates {
		if err := os.MkdirAll(candidate, 0o755); err != nil {
			lastErr = err
			continue
		}
		return candidate, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no candidate directories")
	}
	return "", lastErr
}

// appendUniqueDir adds a directory to the list once.
func appendUniqueDir(dirs *[]string, seen map[string]struct{}, dir string) {
	if dir == "" {
		return
	}
	if _, ok := seen[dir]; ok {
		return
	}
	seen[dir] = struct{}{}
	*dirs = append(*dirs, dir)
}

// copyTigerBeetleDataFile clones or copies a formatted TB file into a temp dir.
func copyTigerBeetleDataFile(t *testing.T, src, dst string) {
	t.Helper()
	if err := tryCloneFile(src, dst); err == nil {
		return
	}
	srcFile, err := os.Open(src)
	if err != nil {
		t.Fatalf("open tigerbeetle base file: %v", err)
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		t.Fatalf("create tigerbeetle data file: %v", err)
	}
	defer dstFile.Close()
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		t.Fatalf("copy tigerbeetle data file: %v", err)
	}
	if err := dstFile.Sync(); err != nil {
		t.Fatalf("sync tigerbeetle data file: %v", err)
	}
}

// tryCloneFile attempts a copy-on-write clone before falling back to full copies.
func tryCloneFile(src, dst string) error {
	if err := cloneFile(src, dst); err == nil {
		return nil
	}
	switch runtime.GOOS {
	case "linux":
		return exec.Command("cp", "--reflink=auto", src, dst).Run()
	default:
		return exec.Command("cp", src, dst).Run()
	}
}

// waitForFile blocks until the path appears or the timeout elapses.
func waitForFile(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat %s: %w", path, err)
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", path)
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
