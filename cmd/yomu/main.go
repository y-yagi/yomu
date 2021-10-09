package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
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
	cfgure       configure.Configure
	itemsPerSite = map[string][]yomu.Item{}
	site         string
	mu           sync.RWMutex
	showFeeds    bool
	updatedOnly  bool
)

const (
	app = "yomu"
)

func init() {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = "/tmp"
	}
	cachePath := filepath.Join(dir, app)
	cfgure = configure.Configure{Name: app}

	f := filepath.Join(cfgure.ConfigDir(), "config.toml")
	if !osext.IsExist(f) {
		c := yomu.Config{Browser: "google-chrome", URLs: map[string]string{}, CachePath: cachePath}
		cfgure.Save(c)
	}
}

func main() {
	exitCode := run(os.Args, os.Stdout, os.Stderr)
	if cfg.URLs != nil && showFeeds {
		cfgure.Save(cfg)
	}
	os.Exit(exitCode)
}

func run(args []string, outStream, errStream io.Writer) (exitCode int) {
	var configureFlag bool
	var unsubscribeFlag bool
	var subscriptionTarget string
	exitCode = 0

	flags := flag.NewFlagSet(app, flag.ExitOnError)
	flags.SetOutput(errStream)
	flags.BoolVar(&configureFlag, "c", false, "configure")
	flags.StringVar(&subscriptionTarget, "s", "", "subscribe feeds from `URL`")
	flags.BoolVar(&unsubscribeFlag, "u", false, "unsubscribe feeds")
	flags.BoolVar(&updatedOnly, "updated-only", false, "show only updated sites")
	flags.Parse(args[1:])

	err := cfgure.Load(&cfg)
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

	if len(subscriptionTarget) != 0 {
		exitCode = subscribe(subscriptionTarget, outStream, errStream)
		return
	}

	if unsubscribeFlag {
		exitCode = unsubscribe(outStream, errStream)
		return
	}

	var wg sync.WaitGroup
	for url := range cfg.URLs {
		wg.Add(1)
		go fetch(url, errStream, outStream, &wg)
	}
	wg.Wait()

	if len(itemsPerSite) == 0 {
		fmt.Fprintln(outStream, "There are no new feeds.")
		return
	}

	if buildGUI(outStream, errStream) == 0 {
		showFeeds = true
	}

	return
}

func editConfig() error {
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "vim"
	}

	return cfgure.Edit(editor)
}

func subscribe(target string, outStream, errStream io.Writer) int {
	s := subscriber.NewSubscriber(cfg, cfgure)
	if err := s.Subscribe(target); err != nil {
		fmt.Fprintf(outStream, "%v\n", err)
		return 1
	} else {
		fmt.Fprint(outStream, "Done!\n")
	}
	return 0
}

func unsubscribe(outStream, errStream io.Writer) int {
	stdio := terminal.Stdio{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}
	u := unsubscriber.NewUnsubscriber(stdio, cfg, cfgure)
	if err := u.Unsubscribe(); err != nil {
		fmt.Fprintf(outStream, "%v\n", err)
		return 1
	} else {
		fmt.Fprint(outStream, "Done!\n")
	}

	return 0
}

func fetch(url string, errStream, outStream io.Writer, wg *sync.WaitGroup) {
	defer wg.Done()

	if os.Getenv("YOMU_DEBUG") != "" {
		fmt.Fprintf(outStream, "'%v' parse start %v\n", url, time.Now())
	}

	var items []yomu.Item
	var err error
	var feed *gofeed.Feed
	var updated bool

	diskcache := diskcache.New(cfg.CachePath)
	fp := gofeed.NewParser()

	cachedVal, ok := diskcache.Get(url)
	if ok {
		b := bytes.NewBuffer(cachedVal)
		resp, err := http.ReadResponse(bufio.NewReader(b), nil)
		if err != nil {
			fmt.Fprintf(errStream, "'%v' read response error: %v\n", url, err)
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(errStream, "'%v' read response error: %v\n", url, err)
			return
		}

		if len(string(body)) == 0 {
			fmt.Fprintf(errStream, "'%v' response body is empty\n", url)
			return
		}

		feed, err = fp.ParseString(string(body))
		if err != nil {
			fmt.Fprintf(errStream, "'%v' parsed error: %v\n", url, err)
			return
		}
	} else {
		fp.Client = &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Second}
		feed, err = fp.ParseURL(url)
		if err != nil {
			fmt.Fprintf(errStream, "'%v' parsed error: %v\n", url, err)
			return
		}
	}

	for _, item := range feed.Items {
		item := yomu.Item{Title: item.Title, Link: item.Link, Description: item.Description}
		items = append(items, item)
	}

	siteTitle := feed.Title
	if len(feed.Items) > 0 {
		if feed.Items[0].PublishedParsed != nil && feed.Items[0].PublishedParsed.UnixNano() > cfg.LastFetched[url] {
			siteTitle = "*" + siteTitle
			updated = true
		} else if feed.Items[0].UpdatedParsed != nil && feed.Items[0].UpdatedParsed.UnixNano() > cfg.LastFetched[url] {
			siteTitle = "*" + siteTitle
			updated = true
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if updatedOnly {
		if updated {
			itemsPerSite[siteTitle] = items
			cfg.LastFetched[url] = time.Now().UnixNano()
		}
	} else {
		itemsPerSite[siteTitle] = items
		cfg.LastFetched[url] = time.Now().UnixNano()
	}

	if os.Getenv("YOMU_DEBUG") != "" {
		fmt.Fprintf(outStream, "'%v' parse end %v\n", url, time.Now())
	}
}

func buildGUI(outStream, errStream io.Writer) int {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		fmt.Fprintf(errStream, "GUI create error: %v\n", err)
		return 1
	}
	defer g.Close()

	g.Cursor = true
	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		fmt.Fprintf(errStream, "Key bindings error: %v\n", err)
		return 1
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		fmt.Fprintf(errStream, "Unexpected error: %v\n", err)
		return 1
	}

	return 0
}
