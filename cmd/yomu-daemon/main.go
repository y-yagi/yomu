package main

import (
	"fmt"
	"io"
	"log/syslog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/robfig/cron"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/yomu"
)

var (
	cfg       yomu.Config
	syslogger *syslog.Writer
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
	err = c.AddFunc("@hourly", func() {
		fetchAll()
	})
	if err != nil {
		fmt.Fprintf(errStream, "Job setting failed: %v\n", err)
		exitCode = 1
		return
	}
	c.Start()

	<-done

	exitCode = 0
	return
}

func fetchAll() {
	var wg sync.WaitGroup
	for url := range cfg.URLs {
		wg.Add(1)
		go fetch(url, &wg)
	}
	wg.Wait()
}

func fetch(url string, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &http.Client{Transport: httpcache.NewTransport(diskcache.New(cfg.CachePath)), Timeout: time.Duration(cfg.Timeout) * time.Second}
	_, err := client.Get(url)
	if err != nil {
		syslogger.Err(fmt.Sprintf("Fetch '%v' error: %v\n", url, err))
		return
	}
}
