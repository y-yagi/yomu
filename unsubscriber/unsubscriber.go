package unsubscriber

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/yomu"
)

type Unsubscriber struct {
	app string
	cfg yomu.Config
}

func NewUnsubscriber(app string, cfg yomu.Config) *Unsubscriber {
	return &Unsubscriber{app: app, cfg: cfg}
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
		Message: "What feeds do you want to unsubscribe to:",
		Options: options,
	}
	survey.AskOne(prompt, &selected)

	for _, key := range selected {
		url := dict[key]
		delete(u.cfg.URLs, url)
	}

	return configure.Save(u.app, u.cfg)
}
