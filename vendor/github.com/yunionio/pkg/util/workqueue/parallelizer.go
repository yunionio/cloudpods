package workqueue

import (
	"sync"

	utilruntime "github.com/yunionio/pkg/util/runtime"
)

type DoWorkPieceFunc func(piece int)

// Parallelize is a very simple framework that allow for parallelizing
// N independent pieces of work.
func Parallelize(workers, pieces int, doWorkPiece DoWorkPieceFunc) {
	toProcess := make(chan int, pieces)
	for i := 0; i < pieces; i++ {
		toProcess <- i
	}
	close(toProcess)

	if pieces < workers {
		workers = pieces
	}

	wg := sync.WaitGroup{}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer utilruntime.HandleCrash()
			defer wg.Done()
			for piece := range toProcess {
				doWorkPiece(piece)
			}
		}()
	}
	wg.Wait()
}
