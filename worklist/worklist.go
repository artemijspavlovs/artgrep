package worklist

// track all files that will be processed

type Entry struct {
	Path string
}

type Worklist struct {
	jobs chan Entry
}

func (w *Worklist) Add(work Entry) {
	w.jobs <- work
}

func (w *Worklist) Next() Entry {
	j := <-w.jobs
	return j
}

func New(bufferSize int) Worklist {
	return Worklist{make(chan Entry, bufferSize)}
}

func NewJob(path string) Entry {
	return Entry{path}
}

// Finalize add empty records to the end of the worklist that indicates to workers that there's noting left to do for
// them to terminate
func (w *Worklist) Finalize(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		w.Add(Entry{""})
	}
}
