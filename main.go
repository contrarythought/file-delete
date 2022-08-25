package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	ROOT = "C:"
)

type Config struct {
	NumThreads int
	Target     string
}

func NewConfig(args []string) (*Config, error) {
	num_threads, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, err
	}
	return &Config{
		NumThreads: num_threads,
		Target:     args[1],
	}, nil
}

type FileMap struct {
	mu          sync.Mutex
	NumFiles    int
	FilesAdd    map[string]int
	FilesDelete map[int]string
}

func NewFileMap() *FileMap {
	return &FileMap{
		FilesAdd:    make(map[string]int),
		FilesDelete: make(map[int]string),
	}
}

// adds file to delete to map
func (f *FileMap) Add(file_path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.NumFiles++
	f.FilesAdd[file_path] = f.NumFiles
	f.FilesDelete[f.NumFiles] = file_path
}

// takes in list of files to delete and deletes them
func (f *FileMap) Delete(files_to_del []int) error {
	for _, i := range files_to_del {
		if err := os.Remove(f.FilesDelete[i]); err != nil {
			fmt.Println("Failed to remove: ", f.FilesDelete[i])
		}
	}
	return nil
}

type WorkerPool struct {
	NumThreads int
	WorkerFunc func(chan string, *FileMap, int, string)
}

// TODO - reads directory starting from path
func readDir(dirChan chan string, file_path string, fm *FileMap, target string) {
	err := filepath.Walk(file_path, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			dirChan <- path
		} else if strings.Contains(path, target) {
			fm.Add(path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func startWorkers(dirChan chan string, fm *FileMap, numThreads int, target string) {
	for i := 0; i < numThreads; i++ {
		go func() {
			for dir := range dirChan {
				readDir(dirChan, dir, fm, target)
			}
		}()
	}
}

func NewWorkerPool(threads int, startWorkers func(chan string, *FileMap, int, string)) *WorkerPool {
	return &WorkerPool{
		NumThreads: threads,
		WorkerFunc: startWorkers,
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage <num_threads> <target_substring>")
		os.Exit(1)
	}
	config, err := NewConfig(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Threads: ", config.NumThreads, " Target: ", config.Target)

	file_map := NewFileMap()
	dirChan := make(chan string)
	wp := NewWorkerPool(config.NumThreads, startWorkers)
	wp.WorkerFunc(dirChan, file_map, config.NumThreads, config.Target)

	// begin adding directories to the channel
	readDir(dirChan, ROOT, file_map, config.Target)
}
