package core

import "sync"

/// 协程池主结构体
type GoWorkers struct {
	// 当前工作者数量
	currentSize int32
	// 限定的工作者数量
	capacity int32
	// 工作者链表头部
	workersHeader *taskWorker
	// 任务缓冲区
	taskBuffer chan func()
	// 任务缓冲区紧急阈值
	taskBufferFlag float32
	// 工作者数量紧急阈值
	workerFlag int32
	// 是否被停止的标记
	isStoped bool
	// 🔒
	mu sync.Mutex
	// 停止的队列
	stopWg sync.WaitGroup
}

func NewGoWorker(capacity, bufferCap int32) *GoWorkers {
	if capacity <= 0 {
		capacity = 1
	}
	if bufferCap <= 0 {
		bufferCap = 1
	}
	gw := &GoWorkers{
		currentSize:    0,
		capacity:       capacity,
		workersHeader:  nil,
		taskBuffer:     make(chan func(), bufferCap),
		taskBufferFlag: float32((bufferCap << 1)) / 3,
		workerFlag:     capacity >> 1,
		isStoped:       false,
	}
	go gw.goWorkerRun()
	return gw
}

func (gw *GoWorkers) CommitTask(task func()) {
	gw.taskBuffer <- task
}

func (gw *GoWorkers) TryCommitTask(task func()) bool {
	select {
	case gw.taskBuffer <- task:
		return true
	default:
		return false
	}
}

func (gw *GoWorkers) Stop() {
	gw.mu.Lock()
	if gw.isStoped {
		gw.mu.Unlock()
		return
	}
	gw.isStoped = true
	close(gw.taskBuffer)
	// 关闭非活动worker
	run := gw.workersHeader
	for run != nil {
		temp := run.next
		run.close()
		run.next = nil
		run = temp
		gw.currentSize -= 1
	}
	gw.stopWg.Add(int(gw.currentSize))
	gw.mu.Unlock()
	gw.stopWg.Wait()
	gw.currentSize = 0
}

func (gw *GoWorkers) GetSize() int32 {
	return gw.currentSize
}

func (gw *GoWorkers) goWorkerRun() {
	for !gw.isStoped {
		task, ok := <-gw.taskBuffer
		if !ok {
			break
		}
		for !gw.lookupWorker(task) {
			if gw.isStoped {
				break
			}
		}
	}

}

func (gw *GoWorkers) lookupWorker(task func()) bool {
	gw.mu.Lock()
	if gw.isStoped {
		gw.mu.Unlock()
		return false
	}
	worker := gw.workersHeader
	if worker != nil {
		gw.workersHeader = worker.next
		worker.next = nil
		gw.mu.Unlock()
		worker.commitTask(task)
		return true
	}
	if gw.currentSize < gw.capacity {
		worker = newtaskWorker(gw)
		gw.currentSize += 1
		worker.commitTask(task)
		gw.mu.Unlock()
		return true
	}
	gw.mu.Unlock()
	return false
}

func (gw *GoWorkers) giveback(tw *taskWorker) {
	gw.mu.Lock()
	defer gw.mu.Unlock()
	if gw.isStoped || (float32(len(gw.taskBuffer)) < gw.taskBufferFlag && gw.currentSize >= gw.workerFlag) {
		tw.close()
		gw.currentSize -= 1
		tw = nil
		if gw.isStoped {
			gw.stopWg.Done()
		}
		return
	}
	if gw.workersHeader != nil {
		tw.next = gw.workersHeader.next
	}
	gw.workersHeader = tw
}
