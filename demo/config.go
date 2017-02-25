package main

import (
	"fmt"
)

type AgentConfig struct {
	ListenPort int `` // default assign when create

	UnixDomainRpc     bool   `` // default false
	UnixDomainRpcFile string `` // default /tmp/${port}
	DisplayName       string `` // default hostname
	JoinOnLoad        string `` // default empty

	EventLength int `` // default 64, length of max event buffer

	Key string `` // key, random base64 std encoding on [16]byte

	// memberlist config
	GossipNodes    int
	IndirectChecks int
}

func DefaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		ListenPort:        *port,
		UnixDomainRpc:     true,
		UnixDomainRpcFile: fmt.Sprintf("/tmp/%d", *port),
		JoinOnLoad:        "",
		EventLength:       64,
		Key:               *flagKey,
		GossipNodes:       7,
		IndirectChecks:    6,
	}
}
