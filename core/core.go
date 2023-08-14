package core

import "fmt"

type GoWorkers struct {
	workerCapacity int32
	workerSize     int32
	taskChannel    chan *workFuncTask
}

type worker struct {
	workerId    int32
	lastWorkAt  int64
	resident    bool
	taskChannel chan (*workFuncTask)
}

type workFuncTask struct {
	task func(any)
	args any
}

func workerFunc(worker *worker) {
	for t := range worker.taskChannel {
		func() {
			defer func() {
				err := recover()
				if err != nil {
					fmt.Println(err)
				}
			}()
			t.task(t.args)
		}()
	}
}

func (gw *GoWorkers) NewGoWorkers(capacity int32) *GoWorkers {
	return &GoWorkers{}
}

func (gw *GoWorkers) CommitFunc(taskFunc func(any), args any) {
	if taskFunc == nil {
		return
	}
	task := &workFuncTask{
		task: taskFunc,
		args: args,
	}
	gw.taskChannel <- task
}

func (gw *GoWorkers) createWorker() {
	newWorker := &worker{}
	go workerFunc(newWorker)
}
