package main

import (
	"flag"
	"fmt"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/serf/serf"
	"log"
	"strings"
	"sync"
	"time"
)

var (
	port    = flag.Int("p", 12356, "port")
	join    = flag.String("join", "127.0.0.1:1234", "init with join exist list")
	flagKey = flag.String("key", "VsuI7f+3+QPDePWCCGhhBQ==", "agent key")
)

type Agent struct {
	serf              *serf.Serf
	eventCh           chan serf.Event
	eventHandlers     map[EventHandler]struct{}
	eventHandlerList  []EventHandler
	eventHandlersLock *sync.Mutex

	stopUnixDomain   chan struct{}
	stopEventHandler chan struct{}
	totalProcess     *sync.WaitGroup

	rpcConnPool *ConnPool

	agentConfig *AgentConfig
}

func NewAgent() *Agent {
	var err error

	ac := DefaultAgentConfig()

	memberconfig := memberlist.DefaultWANConfig()
	memberconfig.BindPort = ac.ListenPort
	memberconfig.BindAddr = GetPublicIP()

	memberconfig.Keyring, err = memberlist.NewKeyring(nil, parmaryKey(ac.Key))
	if err != nil {
		panic("Failed to parse key: " + err.Error())
	}

	memberconfig.GossipNodes = ac.GossipNodes
	memberconfig.IndirectChecks = ac.IndirectChecks

	config := serf.DefaultConfig()
	config.MemberlistConfig = memberconfig

	eventCh := make(chan serf.Event, 64)
	config.EventCh = eventCh

	config.CoalescePeriod = 10 * time.Second
	config.QuiescentPeriod = 2 * time.Second
	config.UserCoalescePeriod = 10 * time.Second
	config.UserQuiescentPeriod = 2 * time.Second

	s, err := serf.Create(config)
	if err != nil {
		panic("Failed to create memberlist: " + err.Error())
	}

	n := s.LocalMember()
	log.Printf("%v %v:%v\n", n.Name, n.Addr, n.Port)

	a := &Agent{
		serf:              s,
		eventCh:           eventCh,
		eventHandlers:     make(map[EventHandler]struct{}),
		eventHandlersLock: new(sync.Mutex),
		stopUnixDomain:    make(chan struct{}),
		stopEventHandler:  make(chan struct{}),
		totalProcess:      new(sync.WaitGroup),
		rpcConnPool:       NewConnPool(),
		agentConfig:       ac,
	}
	handler := new(QueryHandler)
	handler.Response = []byte("ok")
	handler.agent = a
	a.RegisterEventHandler(handler)

	//join implement
	if *join != "127.0.0.1:1234" {
		a.serf.Join([]string{*join}, true)
	}

	var joinlist []string
	for _, v := range strings.Fields(ac.JoinOnLoad) {
		if strings.Contains(v, ":") {
			joinlist = append(joinlist, v)
		} else {
			joinlist = append(joinlist, fmt.Sprintf("%v:%v", v, ac.ListenPort))
		}
	}
	a.serf.Join(joinlist, true)

	return a
}

func (agent *Agent) eventLoop() {
loop:
	for {
		select {
		case e := <-agent.eventCh:
			agent.eventHandlersLock.Lock()
			handlers := agent.eventHandlerList
			agent.eventHandlersLock.Unlock()
			for _, eh := range handlers {
				eh.HandleEvent(e)
			}
		case <-agent.stopEventHandler:
			break loop
		}
	}
	log.Println("event handler stoped")
	agent.totalProcess.Done()
}
