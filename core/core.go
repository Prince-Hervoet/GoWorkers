package core

import (
	"fmt"
	"sync"
	"time"
)

const (
	_WORKER_SIZE_FLAG  = 64
	_WORKER_TIMEOUT    = 5000
	_WORKER_CHECK_SIZE = 16
)

// 工作者类
type GoWorkers struct {
	// goroutine上限
	workerCapacity int32
	// goroutine数量
	workerSize int32
	// 空闲链表长度
	freeSize int32
	// 是否停止了
	isStoped bool
	// 任务缓冲区
	tasksBuffer chan func()
	// worker组成的链表
	workersHead *worker
	workersTail *worker
	workersLock sync.Mutex
	workerCond  sync.Cond
}

func NewGoWorker(capacity, bufferCap int32) *GoWorkers {
	if capacity <= 1 {
		capacity = 1
	}
	if bufferCap <= 1 {
		bufferCap = 1
	}
	ans := &GoWorkers{
		workerCapacity: capacity,
		workerSize:     0,
		freeSize:       0,
		isStoped:       false,
		tasksBuffer:    make(chan func(), bufferCap),
		workersHead:    nil,
		workersTail:    nil,
	}
	ans.workerCond = *sync.NewCond(&ans.workersLock)
	go ans.run()
	go ans.checkExpire()
	return ans
}

func (gw *GoWorkers) Stop() {
	gw.workersLock.Lock()
	defer gw.workersLock.Unlock()
	gw.isStoped = true
	gw.workerCond.Signal()
	close(gw.tasksBuffer)
	gw.workersHead = nil
	gw.workersTail = nil
	gw.freeSize = 0
	gw.workerCapacity = 0
	gw.workerSize = 0
}

func (gw *GoWorkers) WorkerCap() int32 {
	return gw.workerCapacity
}

func (gw *GoWorkers) WorkerSize() int32 {
	return gw.workerSize
}

func (gw *GoWorkers) FreeSize() int32 {
	return gw.freeSize
}

func (gw *GoWorkers) Execute(task func()) bool {
	if gw.isStoped || task == nil {
		return false
	}
	gw.workersLock.Lock()
	if gw.isStoped {
		gw.workersLock.Unlock()
		return false
	}
	gw.workersLock.Unlock()
	select {
	case gw.tasksBuffer <- task:
		return true
	default:
		return false
	}
}

func (gw *GoWorkers) run() {
	for {
		task, ok := <-gw.tasksBuffer
		if !ok {
			break
		}
		w := gw.getWorker()
		if w != nil {
			w.pushTask(task)
			continue
		}

		if gw.createWorker(task) {
			continue
		}

		w = gw.getWorker()
		for w == nil {
			w = gw.getWorker()
		}
		w.pushTask(task)
	}
}

func (gw *GoWorkers) checkExpire() {
	for {
		gw.workersLock.Lock()
		if gw.isStoped {
			break
		}
		for gw.freeSize <= _WORKER_SIZE_FLAG {
			gw.workerCond.Wait()
			if gw.isStoped {
				break
			}
		}
		now := time.Now().UnixMilli()
		run := gw.workersHead
		var prev *worker = nil
		var temp *worker = nil
		for i := 0; i < _WORKER_CHECK_SIZE; i++ {
			if now-run.lastWorkAt >= _WORKER_TIMEOUT {
				temp = run.next
				gw.removeWorker(prev, run)
				run.close()
				run = temp
			} else {
				prev = run
				run = run.next
			}
		}
		fmt.Println("开始清理")
		gw.workersLock.Unlock()
	}
	if gw.isStoped {
		gw.workersLock.Unlock()
	}
}

func (gw *GoWorkers) getWorker() *worker {
	gw.workersLock.Lock()
	defer gw.workersLock.Unlock()
	return gw.getFrontWorker()
}

func (gw *GoWorkers) createWorker(task func()) bool {
	gw.workersLock.Lock()
	if gw.workerSize == gw.workerCapacity {
		gw.workersLock.Unlock()
		return false
	}
	gw.workersLock.Unlock()
	gw.workerSize += 1
	w := newWorker(gw)
	w.start()
	w.pushTask(task)
	return true
}

func (gw *GoWorkers) giveBackWorker(w *worker) {
	gw.workersLock.Lock()
	defer gw.workersLock.Unlock()
	gw.pushBackWorker(w)
	if gw.freeSize > _WORKER_SIZE_FLAG {
		gw.workerCond.Signal()
	}
}

func (gw *GoWorkers) pushBackWorker(w *worker) {
	if gw.workersHead == nil {
		gw.workersHead = w
		gw.workersTail = w
		gw.freeSize += 1
		return
	}
	gw.workersTail.next = w
	gw.workersTail = w
	gw.freeSize += 1
}

func (gw *GoWorkers) removeWorker(prev, w *worker) {
	if prev == nil {
		// 头节点
		gw.workersHead = w.next
		w.next = nil
	} else {
		temp := w.next
		prev.next = temp
		w.next = nil
	}
	if gw.workersHead == nil {
		gw.workersTail = nil
	} else if gw.workersTail == w {
		gw.workersTail = prev
	}
	gw.freeSize -= 1
}

func (gw *GoWorkers) getFrontWorker() *worker {
	if gw.freeSize == 0 {
		return nil
	}
	ans := gw.workersHead
	gw.workersHead = ans.next
	ans.next = nil
	gw.freeSize -= 1
	return ans
}
