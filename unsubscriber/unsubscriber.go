package unsubscriber

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/yomu"
)

type Unsubscriber struct {
	stdio  terminal.Stdio
	cfg    yomu.Config
	cfgure configure.Configure
}

func NewUnsubscriber(stdio terminal.Stdio, cfg yomu.Config, cfgure configure.Configure) *Unsubscriber {
	return &Unsubscriber{cfg: cfg, stdio: stdio, cfgure: cfgure}
}

func (u *Unsubscriber) Unsubscribe() error {
	selected := []string{}
	options := []string{}
	dict := map[string]string{}

	for url, title := range u.cfg.URLs {
		key := "<" + title + "> " + url
		options = append(options, key)
		dict[key] = url
	}

	prompt := &survey.MultiSelect{
		Message:  "What feeds do you want to unsubscribe to:",
		Options:  options,
		PageSize: 20,
	}
	survey.AskOne(prompt, &selected, survey.WithStdio(u.stdio.In, u.stdio.Out, u.stdio.Err))

	for _, key := range selected {
		url := dict[key]
		delete(u.cfg.URLs, url)
	}

	return u.cfgure.Save(u.cfg)
}
