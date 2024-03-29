package subscriber_test

import (
	"reflect"
	"testing"

	"github.com/y-yagi/configure"
	"github.com/y-yagi/yomu"
	"github.com/y-yagi/yomu/subscriber"
)

func TestSubscribe(t *testing.T) {
	cfg := yomu.Config{URLs: map[string]string{}}
	cfgure := configure.Configure{Name: "yomu-test"}
	subscriber := subscriber.NewSubscriber(cfg, cfgure)

	err := subscriber.Subscribe("https://github.com/y-yagi/yomu/commits/master")
	if err != nil {
		t.Fatalf("expected nil, but got \n\n%s\n", err.Error())
	}

	want := map[string]string{"https://github.com/y-yagi/yomu/commits/master.atom": "Recent Commits to yomu:master"}
	if !reflect.DeepEqual(cfg.URLs, want) {
		t.Fatalf("expected \n%s\n\nbut got \n\n%s\n", want, cfg.URLs)
	}
}

func TestSubscribe_nofeeds(t *testing.T) {
	var cfg yomu.Config
	cfgure := configure.Configure{Name: "yomu-test"}
	subscriber := subscriber.NewSubscriber(cfg, cfgure)

	err := subscriber.Subscribe("https://github.com/y-yagi/yomu")
	got := err.Error()
	want := "can't detect any feeds."
	if got != want {
		t.Fatalf("expected \n%s\n\nbut got \n\n%s\n", want, got)
	}
}

func TestSubscribe_invalidURL(t *testing.T) {
	var cfg yomu.Config
	cfgure := configure.Configure{Name: "yomu-test"}
	subscriber := subscriber.NewSubscriber(cfg, cfgure)

	err := subscriber.Subscribe("github.com/y-yagi/yomu")
	got := err.Error()
	want := `URI parser error: parse "github.com/y-yagi/yomu": invalid URI for request`
	if got != want {
		t.Fatalf("expected \n%s\n\nbut got \n\n%s\n", want, got)
	}
}
