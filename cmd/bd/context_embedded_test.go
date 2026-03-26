//go:build embeddeddolt

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

func TestEmbeddedContext(t *testing.T) {
	if os.Getenv("BEADS_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set BEADS_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	bd := buildEmbeddedBD(t)
	dir, _, _ := bdInit(t, bd, "--prefix", "tx")

	t.Run("context_default", func(t *testing.T) {
		cmd := exec.Command(bd, "context")
		cmd.Dir = dir
		cmd.Env = bdEnv(dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bd context failed: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), "embedded") && !strings.Contains(string(out), ".beads") {
			t.Errorf("expected embedded mode or .beads in context output: %s", out)
		}
	})

	t.Run("context_json", func(t *testing.T) {
		cmd := exec.Command(bd, "context", "--json")
		cmd.Dir = dir
		cmd.Env = bdEnv(dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bd context --json failed: %v\n%s", err, out)
		}
		if len(strings.TrimSpace(string(out))) == 0 {
			t.Error("expected non-empty context --json output")
		}
	})

	t.Run("context_explicit_db_uses_target_beads_dir", func(t *testing.T) {
		targetDir, targetBeadsDir, _ := bdInit(t, bd, "--prefix", "cdb")
		sourceDir, _, _ := bdInit(t, bd, "--prefix", "csrc")
		infoOut := bdInfo(t, bd, targetDir)
		const marker = "Database: "
		idx := strings.Index(infoOut, marker)
		if idx < 0 {
			t.Fatalf("bd info output missing %q: %s", marker, infoOut)
		}
		dbPath := strings.TrimSpace(strings.SplitN(infoOut[idx+len(marker):], "\n", 2)[0])
		if dbPath == "" {
			t.Fatalf("parsed empty database path from bd info output: %s", infoOut)
		}
		cmd := exec.Command(bd, "context", "--db", dbPath)
		cmd.Dir = sourceDir
		cmd.Env = bdEnv(sourceDir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bd context --db failed: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), targetBeadsDir) && !strings.Contains(string(out), "/private"+targetBeadsDir) {
			t.Fatalf("expected context output to use target beads dir %q, got: %s", targetBeadsDir, out)
		}
	})
}

func TestEmbeddedContextConcurrent(t *testing.T) {
	if os.Getenv("BEADS_TEST_EMBEDDED_DOLT") != "1" {
		t.Skip("set BEADS_TEST_EMBEDDED_DOLT=1 to run embedded dolt integration tests")
	}
	t.Parallel()

	bd := buildEmbeddedBD(t)
	dir, _, _ := bdInit(t, bd, "--prefix", "xx")

	const numWorkers = 8
	type workerResult struct {
		worker int
		err    error
	}
	results := make([]workerResult, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func(worker int) {
			defer wg.Done()
			r := workerResult{worker: worker}
			cmd := exec.Command(bd, "context")
			cmd.Dir = dir
			cmd.Env = bdEnv(dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				r.err = fmt.Errorf("context (worker %d): %v\n%s", worker, err, out)
			}
			results[worker] = r
		}(w)
	}
	wg.Wait()
	for _, r := range results {
		if r.err != nil {
			t.Errorf("worker %d failed: %v", r.worker, r.err)
		}
	}
}
