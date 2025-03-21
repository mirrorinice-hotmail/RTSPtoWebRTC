//go:build windows

package main

import (
	"log"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/windows/svc"
)

type myService struct{ sys_sigs chan os.Signal }

const serviceName = "rinortsp2web.service"

func (obj *myService) Execute(args []string, req <-chan svc.ChangeRequest, status chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	status <- svc.Status{State: svc.StartPending}

	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	go mainWork()

	loop := true
	for loop {
		select {
		case r := <-req:
			switch r.Cmd {
			case svc.Stop, svc.Shutdown:
				loop = false
				status <- svc.Status{State: svc.StopPending}
			}
		default:
			log.Println("Service is running...")
			time.Sleep(2 * time.Second)
		}
	}

	status <- svc.Status{State: svc.Stopped}
	time.Sleep(2 * time.Second)
	obj.sys_sigs <- syscall.SIGINT
	return
}

func mainByOs() {
	////////////////
	isWinSvc, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Failed to determine if running in an interactive session: %v", err)
	}

	if isWinSvc {
		service := myService{sys_sigs: gSigs}
		log.Printf("%s must be run as a Windows service.", serviceName)
		err = svc.Run(serviceName, &service)
		if err != nil {
			log.Fatalf("Failed to run service: %v", err)
		}
	} else {
		mainWork()
	}
}
