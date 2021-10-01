package main

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"

	strip "github.com/grokify/html-strip-tags-go"
	"github.com/y-yagi/gocui"
)

const (
	mainView    = "main"
	sideView    = "side"
	detailsView = "details"
)

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, cursorLeft); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone, cursorRight); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, open); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'j', gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'k', gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'h', gocui.ModNone, cursorLeft); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'q', gocui.ModNone, quit); err != nil {
		return err
	}

	return g.SetKeybinding("", 'l', gocui.ModNone, cursorRight)
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView(sideView, -1, 0, int(0.2*float32(maxX)), maxY); err != nil {
		v.Title = "Site"
		v.Highlight = true
		v.SelBgColor = gocui.ColorBlue
		v.SelFgColor = gocui.ColorBlack

		for k := range itemsPerSite {
			if len(site) == 0 {
				site = k
			}
			fmt.Fprintln(v, k)
		}
	}

	if v, err := g.SetView(mainView, int(0.2*float32(maxX)), 0, maxX, int(0.8*float32(maxY))); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Feeds"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

		for _, i := range itemsPerSite[site] {
			fmt.Fprintln(v, i.String())
		}
	}

	if v, err := g.SetView("details", int(0.2*float32(maxX)), int(0.8*float32(maxY)), maxX, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Title = "Details"
		v.Highlight = false
		v.SelFgColor = gocui.ColorBlack
		v.Wrap = true
		refreshDetailsView(g)
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func open(g *gocui.Gui, v *gocui.View) error {
	var l string
	var url string
	var err error

	if v == nil {
		if v, err = g.SetCurrentView(mainView); err != nil {
			return err
		}
	}

	_, cy := v.Cursor()
	if l, err = v.Line(cy); err != nil {
		l = ""
	}

	for _, i := range itemsPerSite[site] {
		if l == i.String() {
			url = i.Link
			break
		}
	}

	if err := exec.Command(cfg.Browser, url).Run(); err != nil {
		if err := openByDefault(url); err != nil {
			fmt.Printf("%v\n", err)
			return err
		}
	}

	return nil
}

func openByDefault(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}

	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	var err error

	if v == nil {
		if v, err = g.SetCurrentView(mainView); err != nil {
			return err
		}
	}

	cx, cy := v.Cursor()

	if v.Name() == sideView {
		lineCount := len(strings.Split(v.ViewBuffer(), "\n"))
		if cy+1 == lineCount-2 {
			return nil
		}
	}

	ox, oy := v.Origin()

	cy += 1
	if cy+oy >= len(itemsPerSite[site]) {
		cy = 0
		if err := v.SetOrigin(ox, 0); err != nil {
			return err
		}
	}

	if err := v.SetCursor(cx, cy); err != nil {
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}

	return drawInfoViews(g, v)
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		v = g.Views()[0]
	}

	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	if v.Name() == sideView {
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
		return drawInfoViews(g, v)
	}

	_, maxY := g.Size()

	cy -= 1
	if cy < 0 {
		cy = len(itemsPerSite[site]) - 1
	}

	if cy > maxY {
		if err := v.SetOrigin(ox, cy); err != nil {
			return err
		}
	}

	if err := v.SetCursor(cx, cy); err != nil && oy > 0 {
		if err := v.SetOrigin(ox, oy-1); err != nil {
			return err
		}
	}

	return drawInfoViews(g, v)
}

func drawInfoViews(g *gocui.Gui, v *gocui.View) error {
	var err error

	if v.Name() == sideView {
		// set the language which is used in both main and details view
		setSite(g, v)
		if err = refreshMainView(g, v); err != nil {
			return err
		}

		if err = refreshDetailsView(g); err != nil {
			return err
		}

	}

	if v.Name() == mainView {
		if err = refreshDetailsView(g); err != nil {
			return err
		}
	}

	return nil
}

func setSite(g *gocui.Gui, v *gocui.View) error {
	var l string
	var err error

	if v.Name() == sideView {
		_, cy := v.Cursor()

		if l, err = v.Line(cy); err != nil {
			l = ""
		}

		site = l

		for _, name := range []string{mainView, detailsView} {
			v, _ := g.View(name)
			v.SetCursor(0, 0)
			v.SetOrigin(0, 0)
		}
	}
	return nil
}

func cursorLeft(g *gocui.Gui, view *gocui.View) error {
	v, err := g.SetCurrentView(sideView)
	if err != nil {
		return err
	}

	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy); err != nil {
			return err
		}
	}
	return nil
}

func cursorRight(g *gocui.Gui, view *gocui.View) error {
	v, err := g.SetCurrentView(mainView)
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}

	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy); err != nil {
			return err
		}
	}
	return refreshDetailsView(g)
}

func refreshDetailsView(g *gocui.Gui) error {
	mainView, _ := g.View(mainView)
	_, cy := mainView.Cursor()
	_, oy := mainView.Origin()

	detailsView, _ := g.View(detailsView)
	detailsView.Clear()

	item := itemsPerSite[site][cy+oy]

	fmt.Fprintf(detailsView, "[%s]\n%s", item.Title, strip.StripTags(item.Description))
	return nil
}

func refreshMainView(g *gocui.Gui, v *gocui.View) error {
	var l string
	var err error

	mainView, _ := g.View(mainView)
	_, cy := v.Cursor()
	_, oy := v.Origin()

	if l, err = v.Line(cy + oy); err != nil {
		l = ""
	}

	if len(l) != 0 {
		mainView.Clear()
		for _, r := range itemsPerSite[l] {
			fmt.Fprintln(mainView, r.String())
		}
	}
	return nil
}
