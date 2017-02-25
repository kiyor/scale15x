package main

import (
	"bufio"
	"fmt"
	"github.com/hashicorp/serf/serf"
	"github.com/wsxiaoys/terminal/color"
	"io"
	"log"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type ConnPool struct {
	pool map[string]*net.UnixConn
	*sync.Mutex
}

func NewConnPool() *ConnPool {
	return &ConnPool{
		pool:  make(map[string]*net.UnixConn),
		Mutex: new(sync.Mutex),
	}
}

func (agent *Agent) unixdomain() {
	sock := agent.agentConfig.UnixDomainRpcFile
	if _, err := os.Stat(sock); err == nil {
		log.Println(sock, "exist; remove first")
		os.Remove(sock)
	}
	l, err := net.ListenUnix("unix", &net.UnixAddr{sock, "unix"})
	if err != nil {
		panic(err)
	}
	defer func() {
		l.Close()
		os.Remove(sock)
	}()
	os.Chmod(sock, 0777)
	log.Println("ListenUnix", sock)

	go func() {
		for {
			conn, err := l.AcceptUnix()
			if err != nil {
				break
			}
			if conn != nil {
				go agent.processUnixdomain(conn)
			}
		}
	}()
	<-agent.stopUnixDomain
	log.Println("unixdomain stoped")
	agent.totalProcess.Done()
}

type Msg string

func (m Msg) Byte() []byte {
	return []byte(m)
}

func (m Msg) IsMessage() bool {
	return !m.IsCommand()
}
func (m Msg) IsCommand() bool {
	if !m.IsEmpty() {
		return m[:1] == "/"
	}
	return false
}
func (m Msg) IsEmpty() bool {
	return len(m) == 0
}

func (m Msg) IsQuery() bool {
	return m.IsMessage() && len(strings.Split(string(m), " ")) > 1
}

func (m Msg) ToQuery() (string, string) {
	name := strings.Split(string(m), " ")[0]
	args := strings.TrimLeft(string(m), name)
	args = strings.TrimLeft(args, " ")
	return name, args
}

func (m Msg) Main() string {
	if m.IsCommand() {
		return strings.Split(string(m), " ")[0]
	}
	return ""
}

func (m Msg) Args() []string {
	var args []string
	if m.IsCommand() {
		a := strings.TrimLeft(string(m), m.Main())
		a = strings.TrimLeft(a, " ")
		args = strings.Split(a, " ")
	}
	if len(args) == 1 && len(args[0]) == 0 {
		return []string{}
	}
	return args
}

func (agent *Agent) processUnixdomain(conn *net.UnixConn) {
	key := random(10)
	agent.rpcConnPool.Lock()
	agent.rpcConnPool.pool[key] = conn
	agent.rpcConnPool.Unlock()

	env := newEnv()

	reader := bufio.NewReader(conn)
	for {
		l, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Println(err.Error())
				break
			}
		}
		msg := Msg(strings.TrimRight(l, "\n"))
		go agent.processMsg(conn, env, msg)
	}
	agent.rpcConnPool.Lock()
	delete(agent.rpcConnPool.pool, key)
	agent.rpcConnPool.Unlock()
	conn.Close()
}

func (agent *Agent) processMsg(conn *net.UnixConn, env *Env, msg Msg) {
	if msg.IsEmpty() {
		return
	}
	var mu sync.Mutex
	sendf := func(format string, args ...interface{}) {
		mu.Lock()
		defer mu.Unlock()
		out := fmt.Sprintf(format, args...)
		conn.Write([]byte(out))
	}

	switch msg.Main() {
	case "/env":
		log.Println(len(msg.Args()))
		if len(msg.Args()) == 0 {
			sendf("%v\n", toJson(env.M))
		} else {
			key := msg.Args()[0]
			value := msg.Args()[1]
			env.M[key] = value
			sendf("env \"%s\" set to \"%s\" success\n", key, value)
		}
	case "/join":
		var list []string
		for _, v := range msg.Args() {
			if strings.Contains(v, ":") {
				list = append(list, v)
			} else {
				list = append(list, fmt.Sprintf("%v:%v", v, agent.agentConfig.ListenPort))
			}
		}
		_, err := agent.serf.Join(list, true)
		if err != nil {
			sendf("Fail to join %v, err: %v\n", msg.Args(), err.Error())
		}
	case "/list", "/ll":
		var countTotal, countAlive, countFailed, countLeft, countLeaving int
		members := make(map[string]serf.Member)
		var list []string
		for _, v := range agent.serf.Members() {
			members[v.Name] = v
			list = append(list, v.Name)
		}
		sort.Strings(list)
		for _, k := range list {
			v := members[k]
			countTotal += 1
			var s string
			switch v.Status {
			case serf.StatusAlive:
				countAlive += 1
				s = color.Sprintf("%v %20v:%-6v @{g}%10v@{|}", v.Name, v.Addr, v.Port, v.Status)
			case serf.StatusFailed:
				countFailed += 1
				s = color.Sprintf("%v %20v:%-6v @{r}%10v@{|}", v.Name, v.Addr, v.Port, v.Status)
			case serf.StatusLeft:
				countLeft += 1
				s = color.Sprintf("%v %20v:%-6v @{y}%10v@{|}", v.Name, v.Addr, v.Port, v.Status)
			case serf.StatusLeaving:
				countLeaving += 1
				s = color.Sprintf("%v %20v:%-6v @{b}%10v@{|}", v.Name, v.Addr, v.Port, v.Status)
			}
			sendf("%v\n", s)
		}
		sendf(color.Sprintf("total: %d\t@{g}alive: %d\t@{r}failed: %d\t@{y}left: %d\t@{b}leaving: %d@{|}\n", countTotal, countAlive, countFailed, countLeft, countLeaving))
	case "/list.json":
		for _, v := range agent.serf.Members() {
			sendf("%v\n", toJson(v))
		}
	case "/num", "/ls":
		var countTotal, countAlive, countFailed, countLeft, countLeaving int
		for _, v := range agent.serf.Members() {
			countTotal += 1
			switch v.Status {
			case serf.StatusAlive:
				countAlive += 1
			case serf.StatusFailed:
				countFailed += 1
			case serf.StatusLeft:
				countLeft += 1
			case serf.StatusLeaving:
				countLeaving += 1
			}
		}
		sendf(color.Sprintf("total: %d\t@{g}alive: %d\t@{r}failed: %d\t@{y}left: %d\t@{b}leaving: %d@{|}\n", countTotal, countAlive, countFailed, countLeft, countLeaving))
	case "/state":
		sendf("%v\n", agent.serf.State())
	case "/stats":
		var l []string
		for k := range agent.serf.Stats() {
			l = append(l, k)
		}
		sort.Strings(l)
		for _, v := range l {
			sendf("%20v: %v\n", v, agent.serf.Stats()[v])
		}
	case "/clean":
		for _, v := range agent.serf.Members() {
			if v.Status == serf.StatusFailed {
				err := agent.serf.RemoveFailedNode(v.Name)
				if err != nil {
					sendf("%v\n", err.Error())
				}
			}
		}
	case "/version":
		sendf("%v\n", Version())
	case "/q":
		conn.Close()
	case "/w":
		agent.rpcConnPool.Lock()
		for k := range agent.rpcConnPool.pool {
			sendf("%v\n", k)
		}
		agent.rpcConnPool.Unlock()
	case "/rejoin":
		var list []string
		for _, v := range agent.serf.Members() {
			if v.Status == serf.StatusLeft {
				server := fmt.Sprintf("%v:%v", v.Addr, v.Port)
				list = append(list, server)
			}
		}
		_, err := agent.serf.Join(list, true)
		if err != nil {
			sendf("%v\n", err.Error())
		}
	case "/leave":
		err := agent.serf.Leave()
		if err != nil {
			sendf("%v\n", err.Error())
		}
	case "/say":
		if len(msg.Args()) == 0 {
			sendf("say what?")
			return
		}
		// init query
		name, args := msg.ToQuery()
		qp := agent.serf.DefaultQueryParams()
		qp.RequestAck = false

		// init query message
		m := agent.NewMessage()
		m.SendTime = time.Now()
		log.Println(args)
		m.Body = []byte(args)
		m.Env = env

		// send query message
		agent.serf.Query(name, EncodeMessage(m), qp)

	default:
		if msg.IsCommand() {
			sendf("Command not supported\n")
			return
		}

		// members
		members := make(map[string]bool)
		for _, v := range agent.serf.Members() {
			members[v.Name] = false
		}

		// init query
		name, args := msg.ToQuery()
		qp := agent.serf.DefaultQueryParams()
		qp.RequestAck = true

		// init query message
		m := agent.NewMessage()
		m.SendTime = time.Now()
		m.Body = []byte(args)
		m.Env = env

		// send query message
		resp, err := agent.serf.Query(name, EncodeMessage(m), qp)
		if err != nil {
			sendf("Fail to send query %v, err: %v\n", msg, err.Error())
		}

		// get query message response and output
		var i int
		for r := range resp.ResponseCh() {
			members[r.From] = true
			rm := DecodeMessage(r.Payload)
			dur := time.Since(rm.SendTime)
			var s string
			i += 1
			if dur.Seconds() < 1 {
				s = color.Sprintf("[%4d/%-4d] @{c}%-15v@{|} @{g}%12v@{|}", i, len(members), r.From+":", dur)
			} else if dur.Seconds() < 2 {
				s = color.Sprintf("[%4d/%-4d] @{c}%-15v@{|} @{y}%12v@{|}", i, len(members), r.From+":", dur)
			} else {
				s = color.Sprintf("[%4d/%-4d] @{c}%-15v@{|} @{r}%12v@{|}", i, len(members), r.From+":", dur)
			}
			first := true
			for _, v := range strings.Split(rm.String(), "\n") {
				if first {
					s += fmt.Sprintf("  %v\n", v)
				} else {
					s += fmt.Sprintf("%42v%v\n", " ", v)
				}
				first = false
			}
			sendf("%v", s)
		}
		for k, v := range members {
			if !v {
				sendf("%v\n", color.Sprintf("@{r}%v failed@{|}", k))
			}
		}
	}
}
