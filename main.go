package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

//////////////////////////////

var gSigs chan os.Signal

func main() {
	log.Println("--Start--25.034.01.1")

	gSigs = make(chan os.Signal, 1)
	signal.Notify(gSigs, syscall.SIGINT, syscall.SIGTERM)

	mainByOs()
}

func mainWork() {

	if !setWorkDirectory() {
		return
	}

	gConfig.loadConfig()
	gStreamListInfo.loadList()
	gCctvListMgr.init(&gConfig.Dbms)

	go gCctvListMgr.start()
	go serveHTTP()
	go serveStreams()

	done := make(chan bool, 1)
	go func() {
		sig := <-gSigs
		log.Println("system signal :", sig)
		closeall()
		done <- true
	}()
	log.Println("Awaiting End Signal")

	<-done
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

func setWorkDirectory() bool {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("main()..can't get executable path:", err)
		return false
	}

	exeDir := filepath.Dir(exePath)
	err = os.Chdir(exeDir)
	if err != nil {
		fmt.Println("main()..can't set working path:", err)
		return false
	}

	return true
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
