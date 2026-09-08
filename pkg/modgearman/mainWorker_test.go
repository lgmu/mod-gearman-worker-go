package modgearman

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheckLoads(t *testing.T) {
	if _, err := os.Stat("/proc"); os.IsNotExist(err) {
		t.Skip("skipping test without /proc/")
	}
	disableLogging()
	cfg := config{}
	cfg.loadLimit1 = 999
	cfg.loadLimit5 = 999
	cfg.loadLimit15 = 999

	workerMap := make(map[string]*worker)
	mainworker := newMainWorker(&cfg, []byte("key"), workerMap)

	mainworker.updateLoadAvg()
	passed, _ := mainworker.checkLoads()
	if !passed {
		t.Errorf("loads are to ok, checkload says they are too high")
	}

	cfg.loadLimit1 = 0.01
	cfg.loadLimit5 = 999
	cfg.loadLimit15 = 999

	passed, _ = mainworker.checkLoads()
	if passed {
		t.Errorf("load limit 1 exceeded")
	}

	cfg.loadLimit1 = 999
	cfg.loadLimit5 = 0.01
	cfg.loadLimit15 = 999

	passed, _ = mainworker.checkLoads()
	if passed {
		t.Errorf("load limit 10 exceeded")
	}

	cfg.loadLimit1 = 999
	cfg.loadLimit5 = 999
	cfg.loadLimit15 = 0.01

	passed, _ = mainworker.checkLoads()
	if passed {
		t.Errorf("load limit 15 exceeded")
	}
	setLogLevel(0)
}

func TestAdjustWorkerBottomLevelWaitsForSustainedUnderutilization(t *testing.T) {
	cfg := config{
		minWorker:   1,
		sinkRate:    1,
		idleTimeout: 60,
	}
	workerMap := makeTestWorkerMap(3)
	mainworker := &mainWorker{
		activeWorkers: 0,
		cfg:           &cfg,
		workerMap:     workerMap,
		workerMapLock: new(sync.RWMutex),
		running:       true,
		// Simulate a pool that was created long before utilization dropped.
		idleSince: time.Now().Add(-time.Hour),
	}
	for _, worker := range workerMap {
		worker.mainWorker = mainworker
	}

	// Recovering utilization clears the old timer.
	mainworker.activeWorkers = len(workerMap)
	mainworker.adjustWorkerBottomLevel()
	assert.True(t, mainworker.idleSince.IsZero())

	// The first underutilized sample starts a new stabilization window.
	mainworker.activeWorkers = 0
	mainworker.adjustWorkerBottomLevel()
	assert.Len(t, workerMap, 3)
	assert.WithinDuration(t, time.Now(), mainworker.idleSince, time.Second)

	// Once that window has elapsed, workers are removed at sink-rate.
	mainworker.idleSince = time.Now().Add(-61 * time.Second)
	mainworker.adjustWorkerBottomLevel()
	assert.Len(t, workerMap, 2)
}

func TestAdjustWorkerBottomLevelCalculatesUtilizationBeforeDivision(t *testing.T) {
	cfg := config{
		minWorker:   1,
		sinkRate:    1,
		idleTimeout: 60,
	}
	workerMap := makeTestWorkerMap(10)
	mainworker := &mainWorker{
		activeWorkers: 9,
		cfg:           &cfg,
		workerMap:     workerMap,
		workerMapLock: new(sync.RWMutex),
		running:       true,
		idleSince:     time.Now().Add(-time.Hour),
	}
	for _, worker := range workerMap {
		worker.mainWorker = mainworker
	}

	mainworker.adjustWorkerBottomLevel()

	assert.Len(t, workerMap, 10)
	assert.True(t, mainworker.idleSince.IsZero())
}

func makeTestWorkerMap(count int) map[string]*worker {
	workerMap := make(map[string]*worker, count)
	for i := range count {
		id := fmt.Sprintf("worker-%d", i)
		workerMap[id] = &worker{id: id, what: "check"}
	}

	return workerMap
}
