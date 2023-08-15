package core

import (
	"fmt"
	"time"
)

type worker struct {
	// 所属的池
	owner *GoWorkers
	// 最后一次工作的时间
	lastWorkAt int64
	// 用于接收任务的阻塞channel
	taskChannel chan func()
	// 是否启动
	isRunning bool
	// 用于自耦合链表结构
	next *worker
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
		// 异常处理
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
