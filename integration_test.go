package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

var (
	maxCheckAttempts = 10
)

func TestGitwatch__integration(t *testing.T) {
	// make sure we don't have a cached repo
	os.Remove(".storage/github.com/adamdecaf/gitwatch")

	// build docker image and binary
	out, err := exec.Command("make", "docker").CombinedOutput()
	if err != nil {
		fmt.Printf(`Output:
%s`, string(out))
		t.Fatal(err)
	}

	// run container
	container := exec.Command("docker", "run", "-t", fmt.Sprintf("gitwatch:%s", Version), "-interval", "1s")

	var stdout bytes.Buffer
	container.Stdout = &stdout
	var stderr bytes.Buffer
	container.Stderr = &stderr

	err = container.Start()
	if err != nil {
		container.Process.Kill()
		t.Fatal(err)
	}

	// wait for container to boot and start
	time.Sleep(5 * time.Second)

	// Check that the following lines are present
	clone := "git clone on github.com/adamdecaf/gitwatch"
	fetch := "git fetch on github.com/adamdecaf/gitwatch"

	for i := 0; i < maxCheckAttempts; i++ {
		output := stdout.String()
		if strings.Contains(output, clone) && strings.Contains(output, fetch) {
			// Success, let's cleanup and be done
			container.Process.Kill()
			t.Skip("success")
		}
		time.Sleep(1 * time.Second)
	}

	// Didn't find log lines, either from timeout or mismatch
	t.Fatalf(`Either timed out trying to find output, or didn't match output.
Stdout:
%s
Stderr:
%s`, stdout.String(), stderr.String())
}
