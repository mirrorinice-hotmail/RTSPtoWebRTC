package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("--Start--1701")
	gConfig.loadConfig()
	gStreamListInfo.init(&gConfig.Streams)
	gCctvListMgr.init(&gConfig.Dbms)

	go gCctvListMgr.start()
	go serveHTTP()
	go serveStreams()

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Println("system signal :", sig)
		closeall()
		done <- true
	}()
	log.Println("Awaiting End Signal")

	bContinue := true
	for bContinue {
		select {
		case <-done:
			log.Println("--> end msg")
			bContinue = false
		}
	}
	log.Println("--End--")
}

func closeall() {
	{ //timeout 5 seconds
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := serverHttp.Shutdown(ctx); err != nil {
			log.Fatal("Server forced to shutdown:", err)
		}
	}
	gCctvListMgr.request_stop_and_wait()

}

func restart() {
	fmt.Println("Restarting the program...")

	path, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return
	}

	cmd := exec.Command(path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		fmt.Println("Error restarting the program:", err)
		return
	}

	fmt.Println("Program restarted successfully.")
	os.Exit(0) // Exit the current process.
}
