package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/mmcdole/gofeed"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/gocui"
	"github.com/y-yagi/goext/osext"
	"github.com/y-yagi/yomu"
	"github.com/y-yagi/yomu/subscriber"
	"github.com/y-yagi/yomu/unsubscriber"
)

var (
	cfg          yomu.Config
	itemsPerSite = map[string][]yomu.Item{}
	site         string
	mu           sync.RWMutex
	showFeeds    bool
)

const (
	mainView    = "main"
	sideView    = "side"
	detailsView = "details"
	app         = "yomu"
)

func init() {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = "/tmp"
	}
	cachePath := filepath.Join(dir, app)

	f := filepath.Join(configure.ConfigDir(app), "config.toml")
	if !osext.IsExist(f) {
		c := yomu.Config{Browser: "google-chrome", URLs: map[string]string{}, CachePath: cachePath}
		configure.Save(app, c)
	}
}

func main() {
	exitCode := run(os.Args, os.Stdout, os.Stderr)
	if cfg.URLs != nil && showFeeds {
		configure.Save(app, cfg)
	}
	os.Exit(exitCode)
}

func run(args []string, outStream, errStream io.Writer) (exitCode int) {
	var configureFlag bool
	var unsubscribeFlag bool
	var subscribe string
	exitCode = 0

	flags := flag.NewFlagSet(app, flag.ExitOnError)
	flags.SetOutput(errStream)
	flags.BoolVar(&configureFlag, "c", false, "configure")
	flags.StringVar(&subscribe, "s", "", "subscribe feeds from `URL`")
	flags.BoolVar(&unsubscribeFlag, "u", false, "unsubscribe feeds")
	flags.Parse(args[1:])

	err := configure.Load(app, &cfg)
	if err != nil {
		fmt.Fprintf(errStream, "%v\n", err)
		exitCode = 1
		return
	}

	if cfg.URLs == nil {
		cfg.URLs = map[string]string{}
	}
	if cfg.LastFetched == nil {
		cfg.LastFetched = map[string]int64{}
	}

	if configureFlag {
		if err = editConfig(); err != nil {
			fmt.Fprintf(outStream, "%v\n", err)
			exitCode = 1
		} else {
			fmt.Fprint(outStream, "Done!\n")
		}
		return
	}

	if len(subscribe) != 0 {
		s := subscriber.NewSubscriber(app, cfg)
		if err = s.Subscribe(subscribe); err != nil {
			fmt.Fprintf(outStream, "%v\n", err)
			exitCode = 1
		} else {
			fmt.Fprint(outStream, "Done!\n")
		}
		return
	}

	if unsubscribeFlag {
		u := unsubscriber.NewUnsubscriber(app, cfg)
		if err = u.Unsubscribe(); err != nil {
			fmt.Fprintf(outStream, "%v\n", err)
			exitCode = 1
		} else {
			fmt.Fprint(outStream, "Done!\n")
		}
		return
	}

	var wg sync.WaitGroup
	for url := range cfg.URLs {
		wg.Add(1)
		go fetch(url, errStream, outStream, &wg)
	}
	wg.Wait()

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		fmt.Fprintf(errStream, "GUI create error: %v\n", err)
		exitCode = 1
		return
	}
	defer g.Close()

	g.Cursor = true
	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		fmt.Fprintf(errStream, "Key bindings error: %v\n", err)
		exitCode = 1
		return
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		fmt.Fprintf(errStream, "Unexpected error: %v\n", err)
		exitCode = 1
		return
	}

	showFeeds = true
	return
}

func editConfig() error {
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "vim"
	}

	return configure.Edit(app, editor)
}

func fetch(url string, errStream, outStream io.Writer, wg *sync.WaitGroup) {
	defer wg.Done()

	if os.Getenv("YOMU_DEBUG") != "" {
		fmt.Fprintf(outStream, "'%v' parse start %v\n", url, time.Now())
	}

	var items []yomu.Item

	timeout := cfg.Timeout
	fp := gofeed.NewParser()
	fp.Client = &http.Client{Transport: httpcache.NewTransport(diskcache.New(cfg.CachePath)), Timeout: time.Duration(timeout) * time.Second}
	feed, err := fp.ParseURL(url)
	if err != nil {
		fmt.Fprintf(errStream, "'%v' parsed error: %v\n", url, err)
		return
	}

	for _, item := range feed.Items {
		item := yomu.Item{Title: item.Title, Link: item.Link, Description: item.Description}
		items = append(items, item)
	}

	siteTitle := feed.Title
	if len(feed.Items) > 0 {
		if feed.Items[0].PublishedParsed != nil && feed.Items[0].PublishedParsed.UnixNano() > cfg.LastFetched[url] {
			siteTitle = "*" + siteTitle
		} else if feed.Items[0].UpdatedParsed != nil && feed.Items[0].UpdatedParsed.UnixNano() > cfg.LastFetched[url] {
			siteTitle = "*" + siteTitle
		}
	}

	mu.Lock()
	defer mu.Unlock()
	itemsPerSite[siteTitle] = items
	cfg.LastFetched[url] = time.Now().UnixNano()

	if os.Getenv("YOMU_DEBUG") != "" {
		fmt.Fprintf(outStream, "'%v' parse end %v\n", url, time.Now())
	}
}
