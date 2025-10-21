package patman

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"syscall"
)

// Job represents the parallelizable unit of work happening at line level.
// Each line is distributed to a worker and then handed back to a collector
type Job struct {
	Seq  int64
	Line string
}

type Result struct {
	Seq int64

	// [{match, name}, ...]
	Results [][]string
	Err     error
}

var input string
var index string
var format string
var mem int
var help bool
var exitOnError bool
var workers int
var queueSize int
var pipelines [][]Command
var pipelineNames []string
var delimiter string
var joinDelimiter string
var stdoutBufferSize int

func init() {
	flag.StringVar(&input, "file", "", "input file")
	flag.StringVar(&index, "index", "", "index property used to aggregate logs")
	flag.StringVar(&format, "format", "stdout", "format to be used for output, pipelines are printed in order")
	flag.BoolVar(&help, "help", false, "shows help message")
	flag.BoolVar(&help, "h", false, "shows help message")
	flag.BoolVar(&exitOnError, "exit", true, "terminate execution immediately on first pipeline error")
	flag.IntVar(&mem, "mem", 10, "Buffer size in MB")
	flag.IntVar(&workers, "workers", 0, "number of parallel workers (0 = auto-detect CPU count)")
	flag.IntVar(&queueSize, "queue", 10000, "bounded job queue size for backpressure")
	flag.StringVar(&delimiter, "delimiter", "", "split input into a sequence of lines using a custom delimiter")
	flag.StringVar(&joinDelimiter, "join", "", "join output using a custom delimiter. Writes to stdout")
	flag.IntVar(&stdoutBufferSize, "buffer", 0, "flush stdout in batches to increase performance")
}

func Run() {
	flag.Parse()

	if help {
		flag.Usage()
		usage()
		os.Exit(0)
	}

	for _, raw := range os.Args[1:] {
		var ops []string
		for key := range operators {
			ops = append(ops, key)
		}
		if !regexp.MustCompile("(" + strings.Join(ops, "|") + ")\\(").MatchString(raw) {
			continue
		}

		parser := NewParser(raw)
		cmds, err := parser.Parse()
		if err != nil {
			log.Fatal(err)
		}

		for _, cmd := range cmds {
			if cmd.Name == "name" {
				pipelineNames = append(pipelineNames, cmd.Arg)
			}
		}

		pipelines = append(pipelines, cmds)
	}

	if index != "" {
		indexPipeline := false
		for _, name := range pipelineNames {
			if name == index {
				indexPipeline = true
			}
		}

		if !indexPipeline {
			log.Fatalf("index `%s` must have a matching named pipeline", index)
		}
	}

	scanner := bufio.NewScanner(os.Stdin)

	var f *os.File
	if len(input) > 0 {
		var err error
		f, err = os.Open(input)
		if err != nil {
			log.Fatalf("failed to open input: %v", err)
		}
		scanner = bufio.NewScanner(f)
	}

	print := handleCustomFormatPrint
	for name, p := range printers {
		if name == format {
			print = p
		}
	}

	if joinDelimiter != "" {
		print = handleJoinPrint
	}

	if stdoutBufferSize > 0 {
		print = handleBufferedStdoutPrint
		defer flushBufferedStdout()
	}

	usedMem := mem * 1024 * 1024
	buf := make([]byte, 0, usedMem)
	scanner.Buffer(buf, usedMem)

	if delimiter != "" {
		scanner.Split(ScanDelimiter(delimiter))
	} else {
		scanner.Split(bufio.ScanLines)
	}

	numWorkers := workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-sigChan
		cancel() // trigger graceful shutdown
	}()

	jobsCh := make(chan Job, queueSize)
	resultsCh := make(chan Result, numWorkers*2)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Go(func() {
			worker(ctx, jobsCh, resultsCh)
		})
	}

	wg.Go(func() {
		collector(ctx, resultsCh, print)
	})

	var seq int64
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		case jobsCh <- Job{Seq: seq, Line: scanner.Text()}:
			seq++
		}
	}

	if stdoutBufferSize > 0 {
		flushBufferedStdout()
	}

	// TODO: should be considered while scanning
	if err := scanner.Err(); err != nil {
		log.Printf("scanner error: %v", err)
		cancel()
	}

	wg.Wait()
	if f != nil {
		f.Close()
	}
}

func collector(ctx context.Context, results <-chan Result, print printer) {
	ordering := make(map[int64][][]string)

	var seq int64
	for result := range results {
		select {
		case <-ctx.Done():
			return
		default:
			if result.Err != nil && exitOnError {
				log.Fatalf("error processing line %d: %v", result.Seq, result.Err)
			}

			ordering[result.Seq] = result.Results

			for {
				results, exists := ordering[seq]
				if !exists {
					break
				}

				if index == "" {
					print(results)
				} else {
					buffered := buffer(results)
					if buffered != nil {
						print(buffered)
					}
				}

				// clean up to avoid growing memory usage of ordering
				// buffer in case of many pending pipelines
				delete(ordering, seq)
				seq++
			}
		}
	}
}

func worker(ctx context.Context, jobsCh <-chan Job, resultsCh chan<- Result) {
	for job := range jobsCh {
		select {
		case <-ctx.Done():
			return
		default:
			var results [][]string
			for _, pipeline := range pipelines {
				match, name, err := handle(job.Line, pipeline)
				if err != nil && exitOnError {
					resultsCh <- Result{Seq: job.Seq, Results: nil, Err: err}
					return
				}
				if len(match) > 0 {
					results = append(results, []string{match, name})
				}
			}

			if len(pipelineNames) > 0 {
				sortPipelines(results)
			}

			resultsCh <- Result{Seq: job.Seq, Results: results, Err: nil}
		}
	}
}

func sortPipelines(results [][]string) {
	slices.SortFunc(results, func(a, b []string) int {
		// Unnamed pipelines should be pushed last
		aIndex := -1
		bIndex := -1
		for i, name := range pipelineNames {
			if name == a[1] {
				aIndex = i
			}
			if name == b[1] {
				bIndex = i
			}
		}
		if aIndex < 0 {
			return 1
		}
		if bIndex < 0 {
			return -1
		}
		return aIndex - bIndex
	})
}

func handle(line string, cmds []Command) (string, string, error) {
	cmd := cmds[0]

	match, err := operators[cmd.Name].Operator(line, cmd.Arg)
	if err != nil {
		return "", "", err
	}

	var name string
	if cmd.Name == "name" {
		name = cmd.Arg
	}

	if len(cmds) > 1 {
		return handle(match, cmds[1:])
	}

	return match, name, nil
}

func usage() {
	fmt.Println("Available commands:")
	for name, entry := range operators {
		if entry.Usage == "" {
			continue
		}

		cmd := name
		if entry.Alias != "" {
			cmd = name + ", " + entry.Alias
		}
		fmt.Println("  ", cmd)

		if entry.Usage != "" {
			fmt.Println("       ", entry.Usage)
		}
		if entry.Example != "" {
			fmt.Println("        e.g.", entry.Example)
		}
	}
}
