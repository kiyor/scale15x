package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	flagTest    = flag.Bool("t", false, "test config and exit")
	flagVersion = flag.Bool("v", false, "output version and exit")
)

func init() {
	flag.Parse()
	log.SetFlags(19)
	if *flagTest {
		os.Exit(0)
	}
	if *flagVersion {
		fmt.Println(Version())
		os.Exit(0)
	}
}
func main() {
	agent := NewAgent()
	agent.totalProcess.Add(2)
	go agent.unixdomain()
	go agent.eventLoop()

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

forever:
	for {
		select {
		case s := <-sig:
			if s == syscall.SIGHUP {
				log.Println("reload")
			} else {
				log.Printf("Signal (%d) received, stopping\n", s)
				// 				agent.serf.Leave()
				agent.stopUnixDomain <- struct{}{}
				agent.stopEventHandler <- struct{}{}
				agent.totalProcess.Wait()
				log.Println("stoped")
				break forever
			}
		}
	}
}
