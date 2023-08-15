package core

import (
	"sync"
	"time"
)

const (
	_WORKER_SIZE_FLAG  = 64   // 检查工作者的阈值数量
	_WORKER_TIMEOUT    = 5000 // 工作者过期时间5000ms
	_WORKER_CHECK_SIZE = 16   // 每次检查的工作者数量
)

// 工作者类
type GoWorkers struct {
	// goroutine上限
	workerCapacity int32
	// goroutine数量
	workerSize int32
	// 任务缓存上限
	taskCapacity int32
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
	closeWg     sync.WaitGroup
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
		taskCapacity:   bufferCap,
		isStoped:       false,
		tasksBuffer:    make(chan func(), bufferCap),
		workersHead:    nil,
		workersTail:    nil,
	}
	ans.workerCond = *sync.NewCond(&ans.workersLock)

	// 开启两个协程进行后台任务
	go ans.run()
	go ans.checkExpire()
	return ans
}

func (gw *GoWorkers) Reboot() {
	gw.workersLock.Lock()
	defer gw.workersLock.Unlock()
	if !gw.isStoped {
		return
	}
	gw.isStoped = false
	gw.tasksBuffer = make(chan func(), gw.taskCapacity)
	go gw.run()
	go gw.checkExpire()
}

func (gw *GoWorkers) Stop() {
	gw.workersLock.Lock()
	gw.closeWg.Add(2)

	// 关闭扫描协程
	gw.isStoped = true
	gw.workerCond.Signal()

	// 关闭主处理协程
	close(gw.tasksBuffer)
	gw.workersHead = nil
	gw.workersTail = nil
	gw.freeSize = 0
	gw.workerSize = 0
	gw.workersLock.Unlock()
	gw.closeWg.Wait()
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

func (gw *GoWorkers) ExecuteWait(task func()) {
	if gw.isStoped || task == nil {
		return
	}
	gw.workersLock.Lock()
	if gw.isStoped {
		gw.workersLock.Unlock()
		return
	}
	gw.workersLock.Unlock()
	gw.tasksBuffer <- task
}

func (gw *GoWorkers) run() {
	for {
		task, ok := <-gw.tasksBuffer
		if !ok {
			break
		}

		// 在空闲链表中获取worker
		w := gw.getWorker()
		if w != nil {
			w.pushTask(task)
			continue
		}

		// 如果没有空闲的worker，则尝试创建
		if gw.createWorker(task) {
			continue
		}

		// 如果创建失败，则循环获取空闲worker直到获取到
		w = gw.getWorker()
		for w == nil {
			w = gw.getWorker()
		}
		w.pushTask(task)
	}
	gw.closeWg.Done()
}

func (gw *GoWorkers) checkExpire() {
	for {
		gw.workersLock.Lock()
		if gw.isStoped {
			break
		}

		// 如果数量没有达到阈值，就wait等待唤醒
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

		// 检查16个节点（设定这个值只是为了避免这个任务占用太长的锁）
		for i := 0; i < _WORKER_CHECK_SIZE; i++ {
			if now >= _WORKER_TIMEOUT+run.lastWorkAt {
				temp = run.next
				gw.removeWorker(prev, run)
				run.close()
				run = temp
			} else {
				prev = run
				run = run.next
			}
		}
		gw.workersLock.Unlock()
	}
	if gw.isStoped {
		gw.workersLock.Unlock()
	}
	gw.closeWg.Done()
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
