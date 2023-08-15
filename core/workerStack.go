package core

type workerStack struct {
	workers []*worker
	size    int32
}

func newWorkerStack() *workerStack {
	return &workerStack{
		workers: make([]*worker, 0),
		size:    0,
	}
}

func (ws *workerStack) getSize() int32 {
	return ws.size
}

func (ws *workerStack) push(worker *worker) {
	ws.workers = append(ws.workers, worker)
	ws.size += 1
}

func (ws *workerStack) pop() *worker {
	ans := ws.workers[ws.size-1]
	ws.workers = ws.workers[0 : ws.size-1]
	ws.size -= 1
	return ans
}
