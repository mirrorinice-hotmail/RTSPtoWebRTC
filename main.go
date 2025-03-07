package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("--Start--")
	gConfig.loadConfig()
	gStreamListInfo.Streams = &(gConfig.Streams)

	go cctvlist_mgr_start()
	go serveHTTP()
	go serveStreams()

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Println(sig)

		{ //timeout 5 seconds
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := serverHttp.Shutdown(ctx); err != nil {
				log.Fatal("Server forced to shutdown:", err)
			}
		}
		cctvlist_mgr_stop_and_wait()
		////////////////////
		done <- true
	}()
	log.Println("Awaiting End Signal")
	<-done
	log.Println("--End--")
}
