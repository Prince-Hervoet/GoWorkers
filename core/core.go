package core

import (
	"errors"
	"sync"
)

var (
	TaskNilError       = errors.New("task is nil")
	TaskQueueFullError = errors.New("queue is full")
)

type GoWorkers struct {
	workerCapacity int32
	workerSize     int32
	taskBuffer     chan *workerTask
	freeWorkers    *workerStack
	isStarted      bool
	mu             sync.Mutex
	workersLock    sync.Mutex
}

func NewGoWorkers(capacity, taskBufferCap int32) *GoWorkers {
	return &GoWorkers{
		workerCapacity: capacity,
		workerSize:     0,
		taskBuffer:     make(chan *workerTask, taskBufferCap),
		freeWorkers:    newWorkerStack(),
	}
}

func (gw *GoWorkers) Start() {
	gw.mu.Lock()
	defer gw.mu.Unlock()
	if gw.isStarted {
		return
	}
	gw.isStarted = true
	go gw.submit()
}

func (gw *GoWorkers) Size() int32 {
	return gw.workerSize
}

func (gw *GoWorkers) Execute(task func(any), args any) error {
	if task == nil {
		return TaskNilError
	}
	wft := &workerTask{
		task: task,
		args: args,
	}
	select {
	case gw.taskBuffer <- wft:
		return nil
	default:
		return TaskQueueFullError
	}
}

func (gw *GoWorkers) ExecuteWait(task func(any), args any) {
	if task == nil {
		return
	}
	wft := &workerTask{
		task: task,
		args: args,
	}
	gw.taskBuffer <- wft
}

func (gw *GoWorkers) submit() {
	for {
		task, ok := <-gw.taskBuffer
		if !ok {
			break
		}
		w := gw.getFreeWorker()
		if w != nil {
			w.putTask(task)
			continue
		}

		if gw.createWorker(task) {
			continue
		}

		for w == nil {
			w = gw.getFreeWorker()
		}
		w.putTask(task)
	}

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

func (gw *GoWorkers) removeWorker(worker *worker) {

}
