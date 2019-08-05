package main

import (
	"log"
	"time"
)

func worker() <-chan struct{} {
	out := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)
		out <- struct{}{}
	}()

	return out
}

func main() {
	log.Println("Start")
	r := worker()
	<-r
	log.Println("Done")
}
