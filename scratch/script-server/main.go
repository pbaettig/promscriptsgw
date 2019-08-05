package main

import (
	"bytes"
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/pbaettig/prometheus-scripts/internal/pkg/scripts"
)

type MutexedBuffer struct {
	Buf   bytes.Buffer
	Mutex sync.Mutex
}

func main() {
	ss, _ := scripts.List("/tmp/scripts")
	bgCtx := context.Background()
	wg := new(sync.WaitGroup)

	var buf MutexedBuffer

	for _, sp := range ss {
		wg.Add(1)
		go func(scriptPath string) {
			ctx, cancel := context.WithTimeout(bgCtx, 1*time.Second)
			defer cancel()
			defer wg.Done()

			rc := scripts.RunAsync(ctx, scriptPath, []string{})
			r := <-rc
			// fmt.Printf("Command: %s\nPID: %d, Exited: %v, Run time: %s\n", r.Command, r.State.Pid(), r.State.Exited(), time.Now().Sub(r.StartTime))
			if r.Err != nil {
				log.Printf("ERROR: %s\n", r.Err.Error())
				return
			}
			buf.Mutex.Lock()
			defer buf.Mutex.Unlock()
			r.Stdout.WriteTo(&buf.Buf)
		}(sp)
	}
	wg.Wait()

	log.Printf("listScripts(): %#+v\n", ss)
	buf.Buf.WriteTo(os.Stdout)
	time.Sleep(time.Second)
}
