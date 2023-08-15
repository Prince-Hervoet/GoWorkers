package core

type waitQueue struct {
	tasks    []*workerTask
	size     int32
	capacity int32
	head     int32
	tail     int32
}

func newWaitQueue(capacity int32) *waitQueue {
	return &waitQueue{
		tasks: make([]*workerTask, capacity),
		size:  0,
		head:  0,
		tail:  0,
	}
}

func (wq *waitQueue) offer(task *workerTask) bool {
	if wq.size == wq.capacity {
		return false
	}
	wq.tasks[wq.tail] = task
	wq.tail += 1
	if wq.tail == wq.capacity {
		wq.tail = 0
	}
	wq.size += 1
	return true
}

func (wq *waitQueue) pop() *workerTask {
	if wq.size == 0 {
		return nil
	}
	ans := wq.tasks[wq.head]
	wq.head += 1
	if wq.head == wq.capacity {
		wq.head = 0
	}
	wq.size -= 1
	return ans
}

func (wq *waitQueue) getSize() int32 {
	return wq.size
}
