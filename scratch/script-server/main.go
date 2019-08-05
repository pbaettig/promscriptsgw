package main

import (
	"context"
	"os"
	"time"

	"github.com/pbaettig/script-server/internal/pkg/scripts"
	log "github.com/sirupsen/logrus"
)

// type MutexedBuffer struct {
// 	Buf   bytes.Buffer
// 	Mutex sync.Mutex
// }

func main() {
	log.SetLevel(log.DebugLevel)
	buf := scripts.RunAll(context.Background(), "/tmp/scripts", 1*time.Second)
	buf.WriteTo(os.Stdout)
}

// func mainOld() {
// 	ss, _ := scripts.List("/tmp/scripts")
// 	bgCtx := context.Background()
// 	wg := new(sync.WaitGroup)

// 	var buf MutexedBuffer

// 	for _, sp := range ss {
// 		wg.Add(1)
// 		go func(scriptPath string) {
// 			ctx, cancel := context.WithTimeout(bgCtx, 1*time.Second)
// 			defer cancel()
// 			defer wg.Done()

// 			rc := scripts.RunAsync(ctx, scriptPath, []string{})
// 			r := <-rc

// 			if r.Err != nil {
// 				log.Printf("ERROR: %s\n", r.Err.Error())
// 				return
// 			}

// 			buf.Mutex.Lock()
// 			defer buf.Mutex.Unlock()

// 			r.Stdout.WriteTo(&buf.Buf)
// 		}(sp)
// 	}
// 	wg.Wait()

// 	log.Printf("listScripts(): %#+v\n", ss)
// 	buf.Buf.WriteTo(os.Stdout)
// }
