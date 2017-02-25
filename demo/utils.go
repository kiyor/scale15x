package main

import (
	"encoding/json"
	"log"
	"reflect"
	"runtime"
)

func toJson(i interface{}) string {
	b, _ := json.MarshalIndent(i, "", "  ")
	return string(b)
}

func Must(i interface{}, err error, caller ...int) interface{} {
	if err != nil {
		var pc uintptr
		if len(caller) == 0 {
			pc, _, _, _ = runtime.Caller(1)
			log.Printf("got error for val=%s type=%T func=%s err=%s caller=%d\n", reflect.ValueOf(i).String(), i, runtime.FuncForPC(pc).Name(), err.Error(), 1)
		} else {
			i := 1
			if len(caller) == 2 {
				i = caller[1]
			}
			for j := i; j <= caller[0]; j++ {
				pc, _, _, _ = runtime.Caller(j)
				log.Printf("got error for val=%s type=%T func=%s err=%s caller=%d\n", reflect.ValueOf(i).String(), i, runtime.FuncForPC(pc).Name(), err.Error(), j)
			}
		}
	}
	return i
}
func ErrLog(err error) {
	if err != nil {
		pc, _, _, _ := runtime.Caller(1)
		log.Printf("got error for %s %s", runtime.FuncForPC(pc).Name(), err.Error())
	}
}
