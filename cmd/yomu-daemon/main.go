package main

import (
	"fmt"
	"io"
	"log/syslog"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/mmcdole/gofeed"
	"github.com/robfig/cron"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/yomu"
)

var (
	cfg       yomu.Config
	syslogger *syslog.Writer
	mu        sync.RWMutex
)

const (
	app = "yomu"
)

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func run(args []string, outStream, errStream io.Writer) (exitCode int) {
	err := configure.Load(app, &cfg)
	if err != nil {
		fmt.Fprintf(errStream, "Config file loading failed: %v\n", err)
		exitCode = 1
		return
	}

	syslogger, err = syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, "yomu-daemon")
	if err != nil {
		fmt.Fprintf(errStream, "Syslog writer creating failed: %v\n", err)
		exitCode = 1
		return
	}

	fetchAll()
	done := make(chan bool)

	c := cron.New()
	err = c.AddFunc("*/10 * * * *", func() {
		fetchAll()
	})
	if err != nil {
		fmt.Fprintf(errStream, "Job setting failed: %v\n", err)
		exitCode = 1
		return
	}
	watchConfigFile()
	c.Start()

	<-done

	exitCode = 0
	return
}

func fetchAll() {
	var wg sync.WaitGroup
	mu.Lock()
	urls := cfg.URLs
	mu.Unlock()

	for url := range urls {
		wg.Add(1)
		go fetch(url, &wg)
	}
	wg.Wait()
}

func fetch(url string, wg *sync.WaitGroup) {
	defer wg.Done()

	fp := gofeed.NewParser()
	fp.Client = &http.Client{Transport: httpcache.NewTransport(diskcache.New(cfg.CachePath))}
	_, err := fp.ParseURL(url)
	if err != nil {
		syslogger.Err(fmt.Sprintf("Fetch '%v' error: %v\n", url, err))
		return
	}
}

func watchConfigFile() {
	file := filepath.Join(configure.ConfigDir(app), "config.toml")
	watcher, _ := fsnotify.NewWatcher()

	go func() {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				mu.Lock()
				err := configure.Load(app, &cfg)
				if err != nil {
					syslogger.Err(fmt.Sprintf("Watch config file error: %v\n", err))
				}
				mu.Unlock()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			syslogger.Err(fmt.Sprintf("Watch config file error: %v\n", err))
		}

	}()
	watcher.Add(file)
}
