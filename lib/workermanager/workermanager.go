package workermanager

import (
	"sync"
	"yt/chatbot/lib/utils/log"
)

type WorkerManager struct {
	wg          sync.WaitGroup
	workerCount int
}

var singletonWorkerManagerInstance *WorkerManager = nil
var startOnce sync.Once

func GetInstance() *WorkerManager {

	startOnce.Do(func() {
		singletonWorkerManagerInstance = &WorkerManager{
			workerCount: 0,
		}
	})
	return singletonWorkerManagerInstance
}

func (wm *WorkerManager) StartWorker(task func(), name string) {

	wm.wg.Add(1)
	wm.workerCount++

	log.GetLogger().Trace("Added worker: " + name)

	go func() {
		defer wm.wg.Done()
		task()
		log.GetLogger().Trace("Worker done: " + name)
		wm.workerCount--
	}()
}

func (wm *WorkerManager) WorkerCount() int {
	return wm.workerCount
}

func (wm *WorkerManager) WaitAll() {
	wm.wg.Wait()
}
