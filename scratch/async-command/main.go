package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type execResult struct {
	Command   string
	Stdout    string
	Stderr    string
	Err       error
	State     *os.ProcessState
	StartTime time.Time
}

func asyncExec(parentCtx context.Context, timeout time.Duration, out chan<- execResult, name string, args []string) {
	result := execResult{}

	fullCmd := make([]string, len(args)+1)
	fullCmd[0] = name
	copy(fullCmd[1:], args)
	result.Command = strings.Join(fullCmd, " ")

	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	result.StartTime = time.Now()
	if err := cmd.Run(); err != nil {
		result.Err = err
	}

	result.State = cmd.ProcessState

	out <- result
}

func asyncExec2(ctx context.Context, name string, args []string) <-chan execResult {
	out := make(chan execResult)

	go func() {
		result := execResult{}

		fullCmd := make([]string, len(args)+1)
		fullCmd[0] = name
		copy(fullCmd[1:], args)
		result.Command = strings.Join(fullCmd, " ")

		cmd := exec.CommandContext(ctx, name, args...)
		result.StartTime = time.Now()
		if err := cmd.Run(); err != nil {
			result.Err = err
		}

		result.State = cmd.ProcessState
		out <- result
	}()

	return out
}

func main() {
	bgCtx := context.Background()
	wg := new(sync.WaitGroup)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(bgCtx, 2*time.Second)
			defer cancel()

			r := asyncExec2(ctx, "sh", []string{"-c", fmt.Sprintf("sleep %d; exit 123", rand.Intn(4)+1)})
			rr := <-r
			fmt.Printf("Command: %s\nPID: %d, Exited: %v, Run time: %s\n", rr.Command, rr.State.Pid(), rr.State.Exited(), time.Now().Sub(rr.StartTime))
			if rr.Err != nil {
				fmt.Printf("ERROR: %s\n", rr.Err.Error())
			}
			fmt.Println()
		}()

	}
	wg.Wait()
}
