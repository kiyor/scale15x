package main

import (
	"fmt"
	"github.com/hashicorp/serf/serf"
	"log"
	"strings"
	"sync"
	"time"
)

type EventHandler interface {
	HandleEvent(serf.Event)
}

type QueryHandler struct {
	Response []byte
	Queries  []*serf.Query
	sync.Mutex
	agent *Agent
}

// query listener
func (h *QueryHandler) HandleEvent(e serf.Event) {
	wg := new(sync.WaitGroup)
	query, ok := e.(*serf.Query)
	if !ok {
		return
	}

	h.Lock()
	defer h.Unlock()

	// query backup
	h.Queries = append(h.Queries, query)
	if len(h.Queries) > 100 {
		h.Queries = h.Queries[len(h.Queries)-100:]
	}

	//logging
	m := DecodeMessage(query.Payload)
	log.Println(query.LTime, m.From, query.Name, m, m.Env.M)

	// send to anyone if connected to rpc

	/* disable output for now
	connPool.Lock()
	for _, v := range connPool.pool {
		v.Write([]byte(fmt.Sprintln(query.LTime, query.Name, string(query.Payload))))
	}
	connPool.Unlock()
	*/

	//hook point to process command
	r := h.agent.NewMessage()
	r.Body = h.Response
	r.SendTime = m.SendTime
	r.AckTime = time.Now()

	switch query.Name {
	case "version":
		r.Body = []byte(Version())
	case "/say":
		h.agent.rpcConnPool.Lock()
		for _, v := range h.agent.rpcConnPool.pool {
			fmt.Fprintf(v, "%s %s: %s\n", time.Now().Format(time.Kitchen), m.From, m.String())
		}
		h.agent.rpcConnPool.Unlock()
	case "ping":
		switch m.String() {
		case "all":
			var list []string
			for _, v := range h.agent.serf.Members() {
				list = append(list, v.Addr.String())
			}
			r.Body = []byte(PingSlice(list))
		default:
			r.Body = []byte(PingSlice(strings.Split(m.String(), " ")))
		}
	case "bash":
		res, err := (&Cmd{Cmd: m.String(), Timeout: m.Env.TimeOut() - time.Second}).Exec()
		if err != nil {
			r.Body = []byte(res + err.Error())
		} else {
			r.Body = []byte(res)
		}
	}

	resp := EncodeMessage(r)

	query.Respond(resp)
	wg.Wait()
}

func (a *Agent) RegisterEventHandler(eh EventHandler) {
	a.eventHandlersLock.Lock()
	defer a.eventHandlersLock.Unlock()

	a.eventHandlers[eh] = struct{}{}
	a.eventHandlerList = nil
	for eh := range a.eventHandlers {
		a.eventHandlerList = append(a.eventHandlerList, eh)
	}
}
