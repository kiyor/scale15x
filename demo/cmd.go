package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

func Ping(input, output chan string) error {
	wg := new(sync.WaitGroup)
	for in := range input {
		wg.Add(1)
		go func(in string, output chan string, wg *sync.WaitGroup) {
			out, err := ping(in)
			if err != nil {
				output <- err.Error()
			} else {
				output <- out
			}
			wg.Done()
		}(in, output, wg)
	}
	wg.Wait()
	close(output)
	return nil
}
func PingSlice(in []string) string {
	input := make(chan string)
	output := make(chan string)
	go Ping(input, output)
	for _, v := range in {
		input <- v
	}
	close(input)
	var res string
	for o := range output {
		res += fmt.Sprintln(o)
	}
	return res
}

func ping(target string) (string, error) {
	cmd := fmt.Sprintf("/bin/ping -c 1 %s|grep from", target)
	return (&Cmd{cmd, 2 * time.Second}).Exec()
}

type Cmd struct {
	Cmd     string
	Timeout time.Duration
}

func (c *Cmd) Exec() (string, error) {
	cmd := exec.Command("/bin/sh", "-c", c.Cmd)

	var bufout, buferr bytes.Buffer
	cmd.Stdout = &bufout
	cmd.Stderr = &buferr

	cmd.Start()
	done := make(chan error)
	go func() { done <- cmd.Wait() }()

	timeout := time.After(c.Timeout)

	select {
	case <-timeout:
		cmd.Process.Kill()
		return strings.TrimRight(bufout.String(), "\n"), errors.New("'" + c.Cmd + "' Command timed out")
	case err := <-done:
		if err != nil {
			return strings.TrimRight(bufout.String(), "\n"), err
		}
		if len(buferr.String()) > 0 {
			err = errors.New(buferr.String())
		}
		return strings.TrimRight(bufout.String(), "\n"), err
	}
}
