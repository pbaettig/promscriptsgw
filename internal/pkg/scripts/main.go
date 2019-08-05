package scripts

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

func isExecutable(p string) bool {
	_, err := exec.LookPath(p)
	if err != nil {
		return false
	}

	return true
}

// List searches the provided directory for executables
func List(dir string) ([]string, error) {
	scripts := make([]string, 0)

	ff, err := ioutil.ReadDir(dir)
	if err != nil {
		return scripts, err
	}

	for _, f := range ff {
		fp := path.Join(dir, f.Name())
		if !isExecutable(fp) {
			log.Println("skipping", fp)
			continue
		}

		scripts = append(scripts, fp)
	}

	return scripts, nil
}

// ExecResult is the result of an asynchronously executed script
type ExecResult struct {
	Command   string
	Stdout    bytes.Buffer
	Stderr    bytes.Buffer
	Err       error
	State     *os.ProcessState
	StartTime time.Time
}

// RunAsync executes the provided executable respecting the passed context
func RunAsync(ctx context.Context, name string, args []string) <-chan ExecResult {
	out := make(chan ExecResult)

	go func() {
		result := ExecResult{}

		fullCmd := make([]string, len(args)+1)
		fullCmd[0] = name
		copy(fullCmd[1:], args)
		result.Command = strings.Join(fullCmd, " ")

		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Stdout = &result.Stdout
		cmd.Stderr = &result.Stderr

		result.StartTime = time.Now()
		if err := cmd.Run(); err != nil {
			result.Err = err
		}

		result.State = cmd.ProcessState
		out <- result
	}()

	return out
}
