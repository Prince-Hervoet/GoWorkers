package core

import (
	"errors"
	"sync"
)

type GoWorkers struct {
	workerCapacity int32
	workerSize     int32
	freeWorkers    *workerStack
	workersLock    sync.Mutex
}

func (gw *GoWorkers) NewGoWorkers(capacity int32) *GoWorkers {
	return &GoWorkers{
		workerCapacity: capacity,
		workerSize:     0,
		freeWorkers:    newWorkerStack(),
	}
}

func (gw *GoWorkers) Execute(task func(any), args any) {
	if task == nil {
		return
	}
	wft := &workerTask{
		task: task,
		args: args,
	}
	gw.submit(wft)
}

func (gw *GoWorkers) submit(task *workerTask) error {
	w := gw.getFreeWorker()
	if w != nil {
		w.putTask(task)
		return nil
	}

	if gw.createWorker(task) {
		return nil
	}
	return errors.New("pool is full")
}

func (gw *GoWorkers) getFreeWorker() *worker {
	gw.workersLock.Lock()
	defer gw.workersLock.Unlock()
	if gw.freeWorkers.getSize() == 0 {
		return nil
	}
	return gw.freeWorkers.pop()
}

func (gw *GoWorkers) createWorker(task *workerTask) bool {
	gw.workersLock.Lock()
	defer gw.workersLock.Unlock()
	if gw.workerSize == gw.workerCapacity {
		return false
	}
	nw := newWorker(gw)
	nw.start()
	nw.putTask(task)
	gw.workerSize += 1
	return true
}

func (gw *GoWorkers) givebackWorker(w *worker) {
	gw.workersLock.Lock()
	defer gw.workersLock.Unlock()
	gw.freeWorkers.push(w)
}
