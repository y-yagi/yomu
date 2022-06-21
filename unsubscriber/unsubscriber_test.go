package unsubscriber_test

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/yomu"
	"github.com/y-yagi/yomu/unsubscriber"
)

func TestUnsubscribe(t *testing.T) {
	t.Skip()

	c, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	cfg := yomu.Config{URLs: map[string]string{"https://github.com/y-yagi/yomu/commits/master.atom": "Recent Commits to yomu:master"}}
	cfgure := configure.Configure{Name: "yomu-test"}
	cfgure.Save(cfg)

	stdio := terminal.Stdio{In: c.Tty(), Out: c.Tty(), Err: c.Tty()}
	unsubscriber := unsubscriber.NewUnsubscriber(stdio, cfg, cfgure)
	go func() {
		unsubscriber.Unsubscribe()
	}()

	go func() {
		c.ExpectEOF()
	}()

	time.Sleep(time.Second)
	c.SendLine("\x1b[C")
	time.Sleep(time.Second)

	cfgure.Load(&cfg)
	want := map[string]string{}
	if !reflect.DeepEqual(cfg.URLs, want) {
		t.Fatalf("expected \n%s\n\nbut got \n\n%s\n", want, cfg.URLs)
	}
}
