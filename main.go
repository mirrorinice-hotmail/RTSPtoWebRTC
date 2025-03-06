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
	gConfig.loadConfig()
	gStreamListInfo.Streams = &(gConfig.Streams)

	cctvlist_mgr_done_sig := make(chan struct{}, 1)
	go cctvlist_mgr(cctvlist_mgr_done_sig)
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
		{
			cctvlist_mgr_stop()
			<-cctvlist_mgr_done_sig
		}
		done <- true
	}()
	log.Println("Server Start Awaiting Signal")
	<-done
	log.Println("Exiting")
}
