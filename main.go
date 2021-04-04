package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/mmcdole/gofeed"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/gocui"
	"github.com/y-yagi/goext/osext"
	"github.com/y-yagi/yomu/subscriber"
	"github.com/y-yagi/yomu/unsubscriber"
	"github.com/y-yagi/yomu/utils"
)

var (
	cfg          utils.Config
	itemsPerSite = map[string][]utils.Item{}
	site         string
)

const (
	mainView    = "main"
	sideView    = "side"
	detailsView = "details"
	app         = "yomu"
)

func init() {
	f := filepath.Join(configure.ConfigDir(app), "config.toml")
	if !osext.IsExist(f) {
		c := utils.Config{Browser: "google-chrome", URLs: map[string]string{}}
		configure.Save(app, c)
	}
}

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func run(args []string, outStream, errStream io.Writer) (exitCode int) {
	var configureFlag bool
	var unsubscribeFlag bool
	var subscribe string
	exitCode = 0

	flags := flag.NewFlagSet(app, flag.ExitOnError)
	flags.SetOutput(errStream)
	flags.BoolVar(&configureFlag, "c", false, "configure")
	flags.StringVar(&subscribe, "s", "", "subscribe an `URL`")
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
		go fetch(url, errStream, &wg)
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

	return
}

func editConfig() error {
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "vim"
	}

	return configure.Edit(app, editor)
}

func fetch(url string, errStream io.Writer, wg *sync.WaitGroup) {
	defer wg.Done()

	var items []utils.Item

	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL(url)

	for _, item := range feed.Items {
		item := utils.Item{Title: item.Title, Link: item.Link, Description: item.Description}
		items = append(items, item)
	}

	itemsPerSite[feed.Title] = items
}
