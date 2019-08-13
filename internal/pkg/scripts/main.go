package scripts

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
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

// MutexedBuffer wraps a Buffer and Mutex object
type MutexedBuffer struct {
	Buf   bytes.Buffer
	Mutex sync.Mutex
}

// RunAll runs all the scripts and returns
// a  Buffer with the collected Stdouts
func RunAll(ctx context.Context, scripts []string, scriptTimeout time.Duration) bytes.Buffer {
	wg := new(sync.WaitGroup)

	var mbuf MutexedBuffer

	for _, sp := range scripts {
		wg.Add(1)
		go func(scriptPath string) {
			slog := log.WithFields(log.Fields{
				"script": scriptPath,
			})

			ctx, cancel := context.WithTimeout(ctx, scriptTimeout)
			defer cancel()
			defer wg.Done()

			slog.Debugf("starting")
			rc := RunAsync(ctx, scriptPath, []string{})
			r := <-rc

			if r.Err != nil {
				slog.Error(r.Err.Error())
				return
			}

			mbuf.Mutex.Lock()
			defer mbuf.Mutex.Unlock()
			r.Stdout.WriteTo(&mbuf.Buf)
		}(sp)
	}

	wg.Wait()
	return mbuf.Buf
}
