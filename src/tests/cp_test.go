package tests

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func mobincusBinary(t *testing.T) string {
	t.Helper()
	if b := os.Getenv("MOBINCUS_BINARY"); b != "" {
		return b
	}
	b, err := exec.LookPath("docker")
	if err == nil {
		return b
	}
	return "../mobincus"
}

func runMobincus(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(mobincusBinary(t), args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runMobincusOK(t *testing.T, args ...string) string {
	t.Helper()
	out, err := runMobincus(t, args...)
	if err != nil {
		t.Fatalf("mobincus %v failed: %v\n%s", args, err, out)
	}
	return out
}

func mkContainer(t *testing.T, image, cmd string) string {
	t.Helper()
	cid := strings.TrimSpace(runMobincusOK(t, "run", "-d", image, "sh", "-c", cmd))
	runMobincusOK(t, "wait", cid)
	return cid
}

func rmContainer(t *testing.T, cid string) {
	t.Helper()
	runMobincus(t, "rm", "-f", cid)
}

func tarFileNames(t *testing.T, data string) []string {
	t.Helper()
	buf := bytes.NewBufferString(data)
	gr, err := gzip.NewReader(buf)
	if err != nil {
		// try uncompressed tar
		tr := tar.NewReader(bytes.NewBufferString(data))
		var names []string
		for {
			hdr, err := tr.Next()
			if err != nil {
				break
			}
			names = append(names, hdr.Name)
		}
		return names
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	var names []string
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		names = append(names, hdr.Name)
	}
	return names
}

func TestCpFileFromContainer(t *testing.T) {
	cid := mkContainer(t, "alpine", "echo hello-world > /test_cp_file.txt")
	defer rmContainer(t, cid)

	dst := filepath.Join(t.TempDir(), "copied_file.txt")
	runMobincusOK(t, "cp", cid+":/test_cp_file.txt", dst)

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(data)); got != "hello-world" {
		t.Fatalf("expected 'hello-world', got %q", got)
	}
}

func TestCpDirectoryFromContainer(t *testing.T) {
	cid := mkContainer(t, "alpine", "mkdir -p /cp_dir/sub && echo inner > /cp_dir/sub/file.txt")
	defer rmContainer(t, cid)

	dst := t.TempDir()
	runMobincusOK(t, "cp", cid+":/cp_dir", dst)

	// Destination is an existing directory: contents of /cp_dir go into it.
	got, err := os.ReadFile(filepath.Join(dst, "sub", "file.txt"))
	if err != nil {
		found := []string{}
		filepath.Walk(dst, func(p string, _ os.FileInfo, _ error) error {
			rel, _ := filepath.Rel(dst, p)
			found = append(found, rel)
			return nil
		})
		t.Fatalf("file not found. Walk of %s: %v", dst, found)
	}
	if strings.TrimSpace(string(got)) != "inner" {
		t.Fatalf("expected 'inner', got %q", string(got))
	}
}

func TestCpFileToContainer(t *testing.T) {
	cid := mkContainer(t, "alpine", "mkdir -p /cp_to")
	defer rmContainer(t, cid)

	src := filepath.Join(t.TempDir(), "host_file.txt")
	if err := os.WriteFile(src, []byte("from-host"), 0644); err != nil {
		t.Fatal(err)
	}

	runMobincusOK(t, "cp", src, cid+":/cp_to/host_file.txt")

	// Verify by copying back and parsing tar
	out := runMobincusOK(t, "cp", cid+":/cp_to/host_file.txt", "-")
	names := tarFileNames(t, out)
	if len(names) == 0 || names[0] != "host_file.txt" {
		t.Fatalf("expected tar to contain host_file.txt, got: %v", names)
	}
}

func TestCpDirectoryToContainer(t *testing.T) {
	cid := mkContainer(t, "alpine", "mkdir -p /cp_to_dir")
	defer rmContainer(t, cid)

	src := filepath.Join(t.TempDir(), "host_dir")
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "data.txt"), []byte("nested-data"), 0644); err != nil {
		t.Fatal(err)
	}

	runMobincusOK(t, "cp", src, cid+":/cp_to_dir/host_dir")

	out := runMobincusOK(t, "cp", cid+":/cp_to_dir/host_dir/nested/data.txt", "-")
	names := tarFileNames(t, out)
	if len(names) == 0 || names[0] != "data.txt" {
		t.Fatalf("expected tar to contain data.txt, got: %v", names)
	}
}

func TestCpToStdout(t *testing.T) {
	cid := mkContainer(t, "alpine", "echo stdout-test > /test_stdout.txt")
	defer rmContainer(t, cid)

	out := runMobincusOK(t, "cp", cid+":/test_stdout.txt", "-")
	names := tarFileNames(t, out)
	if len(names) == 0 || names[0] != "test_stdout.txt" {
		t.Fatalf("expected tar to contain test_stdout.txt, got: %v", names)
	}
}

func TestCpNotFound(t *testing.T) {
	cid := mkContainer(t, "alpine", "echo exists > /real_file.txt")
	defer rmContainer(t, cid)

	_, err := runMobincus(t, "cp", cid+":/nonexistent_file", t.TempDir())
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestCpLocalOnly(t *testing.T) {
	_, err := runMobincus(t, "cp", "/local/src", "/local/dst")
	if err == nil {
		t.Fatal("expected error for local-only paths")
	}
}

func TestCpNameHasColon(t *testing.T) {
	cid := mkContainer(t, "alpine", "echo colon-test > '/tmp/te:s:t'")
	defer rmContainer(t, cid)

	dst := filepath.Join(t.TempDir(), "te:s:t")
	runMobincusOK(t, "cp", cid+":/tmp/te:s:t", dst)

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(data)); got != "colon-test" {
		t.Fatalf("expected 'colon-test', got %q", got)
	}
}

func TestCpSpecialFiles(t *testing.T) {
	cid := mkContainer(t, "alpine", "echo special-test > /etc/test-hostname")
	defer rmContainer(t, cid)

	dst := filepath.Join(t.TempDir(), "test-hostname")
	runMobincusOK(t, "cp", cid+":/etc/test-hostname", dst)

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(data)); got != "special-test" {
		t.Fatalf("expected 'special-test', got %q", got)
	}
}
