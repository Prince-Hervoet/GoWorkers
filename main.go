package main

import (
	"fmt"
	"goworkers/core"
	"time"
)

func main() {
	gw := core.NewGoWorker(100, 500)
	for i := 0; i < 100; i++ {
		a := i
		gw.Execute(func() {
			fmt.Print("test ...")
			fmt.Println(a)
		})
	}
	time.Sleep(1 * time.Second)
	fmt.Println("===========================================")
	for i := 0; i < 100; i++ {
		a := i
		gw.Execute(func() {
			fmt.Print("test ...")
			fmt.Println(a)
		})
	}

	time.Sleep(1 * time.Second)

	gw.Stop()

	fmt.Println("结束了。。。。")
	for {
	}
}
