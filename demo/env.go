package main

import (
	"sync"
	"time"
)

type Env struct {
	M map[string]string
	sync.Mutex
}

func newEnv() *Env {
	env := Env{
		M: make(map[string]string),
	}
	env.M["timeout"] = "5s"
	return &env
}

func (e *Env) TimeOut() time.Duration {
	e.Lock()
	defer e.Unlock()
	if val, ok := e.M["timeout"]; ok {
		if timeout, err := time.ParseDuration(val); err != nil {
			return timeout
		}
	}
	return 5 * time.Second
}

func (e *Env) Set(key, value string) {
	e.Lock()
	e.M[key] = value
	e.Unlock()
}
