package core

import (
	"fmt"
	"time"
)

type worker struct {
	owner       *GoWorkers
	lastWorkAt  int64
	taskChannel chan func()
	isRunning   bool
	next        *worker
}

func newWorker(owner *GoWorkers) *worker {
	return &worker{
		owner:       owner,
		lastWorkAt:  time.Now().UnixMilli(),
		taskChannel: make(chan func()),
	}
}

func (w *worker) start() {
	if w.isRunning {
		return
	}
	w.isRunning = true
	go w.workerRun()
}

func (w *worker) close() {
	if !w.isRunning {
		return
	}
	w.isRunning = false
	close(w.taskChannel)
}

func (w *worker) pushTask(task func()) {
	w.taskChannel <- task
}

func (w *worker) workerRun() {
	solve := func(task func()) {
		defer func() {
			err := recover()
			if err != nil {
				fmt.Println(err)
			}
		}()
		task()
	}
	for {
		task, ok := <-w.taskChannel
		if !ok {
			break
		}
		solve(task)
		w.lastWorkAt = time.Now().UnixMilli()
		w.owner.giveBackWorker(w)
	}
}
