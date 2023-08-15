package main

import (
	"fmt"
	"goworkers/core"
	"time"
)

func main() {
	gw := core.NewGoWorkers(1000, 500)
	gw.Start()
	for i := 0; i < 5; i++ {
		gw.Execute(func(a any) {
			fmt.Println("text")
		}, nil)
		time.Sleep(1 * time.Second)
	}
	time.Sleep(1 * time.Second)
	fmt.Print("size: ")
	fmt.Println(gw.Size())
	for {
	}
}
