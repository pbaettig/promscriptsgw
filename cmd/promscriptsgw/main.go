package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pbaettig/promscriptsgw/internal/app"
	"github.com/pbaettig/promscriptsgw/internal/cfg"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/sirupsen/logrus"
)

func startHTTPServer(wg *sync.WaitGroup) *http.Server {
	srv := &http.Server{Addr: cfg.ListenAddr}

	http.Handle(cfg.MetricsURL, promhttp.HandlerFor(app.Registry, promhttp.HandlerOpts{
		ErrorLog:      log.StandardLogger(),
		ErrorHandling: 1,
	}))

	wg.Add(1)
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %s", err)
		}
		wg.Done()
	}()

	return srv
}

func signalHandler(sigs <-chan os.Signal, loopCancel context.CancelFunc, server *http.Server) {
	sig := <-sigs
	log.Infof("signal \"%s\" received", sig.String())
	loopCancel()

	log.Debug("shutting down HTTP server")
	err := server.Shutdown(context.Background())
	if err != nil {
		log.Debug(err.Error())
	}

}

func main() {
	var (
		flagVersion bool
	)

	flag.BoolVar(&flagVersion, "version", false, "display version information")

	err := cfg.FromCommandline()

	if flagVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if err != nil {
		log.Fatal(err.Error())
	}

	// set up cancellable context for the collection loop
	clCtx, clCancel := context.WithCancel(context.Background())
	defer clCancel()

	// waitGroup to wait for the HTTP server to exit
	srvWait := new(sync.WaitGroup)
	srv := startHTTPServer(srvWait)

	// set up signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go signalHandler(sigs, clCancel, srv)

	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Infof("looking for executable scripts in \"%s\"", cfg.ScriptDir)

	// Start metrics collection loop
	app.CollectionLoop(clCtx)

	log.Infof("listening for requests on %s", cfg.ListenAddr)
	srvWait.Wait()
	log.Info("goodbye!")
}
