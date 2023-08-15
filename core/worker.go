package core

import (
	"fmt"
	"time"
)

type worker struct {
	owner       *GoWorkers
	lastWorkAt  int64
	taskChannel chan (*workerTask)
}

type workerTask struct {
	task func(any)
	args any
}

func newWorker(gw *GoWorkers) *worker {
	return &worker{
		owner:       gw,
		lastWorkAt:  time.Now().UnixMilli(),
		taskChannel: make(chan *workerTask),
	}
}

func (w *worker) start() {
	go w.workerRun()
}

func (w *worker) putTask(task *workerTask) {
	w.taskChannel <- task
}

func (w *worker) workerRun() {
	run := func(task *workerTask) {
		defer func() {
			err := recover()
			if err != nil {
				fmt.Println(err)
			}
		}()
		task.task(task.args)
	}
	defer func() {
		err := recover()
		if err != nil {
			fmt.Println(err)
		}
		w.owner.removeWorker(w)
	}()
	for {
		task, ok := <-w.taskChannel
		if !ok {
			break
		}
		w.lastWorkAt = time.Now().UnixMilli()
		run(task)
		w.owner.givebackWorker(w)
	}
}
