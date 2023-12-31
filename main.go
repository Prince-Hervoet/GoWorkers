package main

import (
	"fmt"
	"goworkers/core"
	"time"
)

func main() {
	gw := core.NewGoWorker(100, 100)
	gw.CommitTask(func() {
		fmt.Println("12")
	})
	gw.CommitTask(func() {
		fmt.Println("23")
	})
	gw.CommitTask(func() {
		fmt.Println("34")
	})
	time.Sleep(1 * time.Second)
	gw.CommitTask(func() {
		fmt.Println("45")
	})
	for {
		time.Sleep(2 * time.Second)
		fmt.Println(gw.GetSize())
	}
}
