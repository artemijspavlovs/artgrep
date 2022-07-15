package main

import (
	"artgrep/worker"
	"artgrep/worklist"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/alexflint/go-arg"
)

// recursively checks a directory, retrieves all file paths and adds them to a worklist
func discoverFilePathsInDir(wl *worklist.Worklist, path string) {
	entries, err := os.ReadDir(path)

	if err != nil {
		fmt.Println("Readdir error:", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			nextPath := filepath.Join(path, entry.Name())
			// recursively calling the function
			discoverFilePathsInDir(wl, nextPath)
		} else {
			wl.Add(worklist.NewJob(filepath.Join(path, entry.Name())))
		}
	}
}

// struct for command line arguments
var args struct {
	SearchTerm string `arg:"positional,required"`
	SearchDir  string `arg:"positional"`
}

func main() {
	// using github.com/alexflint/go-arg, ensure that CLI arguments are correct
	arg.MustParse(&args)

	// create new workers
	var workersWg sync.WaitGroup
	numWorkers := 10

	// create the worklist
	wl := worklist.New(100)

	// channel workers will write to
	results := make(chan worker.Result, 100)

	workersWg.Add(1)

	// goroutine that goes through the directories and finds file paths
	go func() {
		defer workersWg.Done()
		discoverFilePathsInDir(&wl, args.SearchDir)
		wl.Finalize(numWorkers)
	}()

	// spawn workers
	for i := 0; i < numWorkers; i++ {
		workersWg.Add(1)
		go func() {
			defer workersWg.Done()
			for {
				workEntry := wl.Next()
				if workEntry.Path != "" {
					workerResult := worker.FindInFile(workEntry.Path, args.SearchTerm)
					if workerResult != nil {
						for _, r := range workerResult.Inner {
							results <- r
						}
					}
				} else {
					return
				}
			}
		}()
	}

	// wait for workers waitgroup to finish while printing the results
	blockWorkersWg := make(chan struct{})
	go func() {
		workersWg.Wait()
		close(blockWorkersWg)
	}()

	// display results
	var displayResultsWg sync.WaitGroup
	displayResultsWg.Add(1)
	go func() {
		for {
			select {
			case r := <-results:
				fmt.Printf("found in:%v[@%v]:\t%v\n", r.Path, r.LineNum, r.Line)
			// once all results are printed - finish
			case <-blockWorkersWg:
				if len(results) == 0 {
					displayResultsWg.Done()
					return
				}
			}
		}
	}()
	displayResultsWg.Wait()
}
