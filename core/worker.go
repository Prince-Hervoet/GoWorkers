package core

import (
	"fmt"
	"time"
)

var workerIdInc = uint64(1)

type taskWorker struct {
	workerId     uint64
	lastUpdateAt int64
	accpetBuffer chan func()
	next         *taskWorker
	owner        *GoWorkers
}

func newTaskWorker(gw *GoWorkers) *taskWorker {
	workerIdInc += 1
	tw := &taskWorker{
		workerId:     workerIdInc,
		lastUpdateAt: time.Now().UnixMilli(),
		accpetBuffer: make(chan func()),
		next:         nil,
		owner:        gw,
	}
	go tw.taskWorkerRun()
	return tw
}

func (tw *taskWorker) taskWorkerRun() {
	for {
		task, ok := <-tw.accpetBuffer
		if !ok {
			break
		}
		func() {
			defer func() {
				if err := recover(); err != nil {
					fmt.Print("task error - workerId-")
					fmt.Print(tw.workerId)
					fmt.Print(": ")
					fmt.Println(err)
				}
			}()
			task()
		}()
		tw.lastUpdateAt = time.Now().UnixMilli()
		tw.owner.giveback(tw)
	}
}

func (tw *taskWorker) commitTask(task func()) {
	tw.accpetBuffer <- task
}

func (tw *taskWorker) close() {
	close(tw.accpetBuffer)
	tw.next = nil
	tw.owner = nil
}
