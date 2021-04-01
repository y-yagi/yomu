package subscriber

import (
	"fmt"
	"net/url"

	"github.com/AlecAivazis/survey/v2"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/rssfinder"
	"github.com/y-yagi/yomu/utils"
)

type Subscriber struct {
	app string
	cfg utils.Config
}

func NewSubscriber(app string, cfg utils.Config) *Subscriber {
	return &Subscriber{app: app, cfg: cfg}
}

func (s *Subscriber) Subscribe(rawurl string) error {
	_, err := url.ParseRequestURI(rawurl)
	if err != nil {
		return fmt.Errorf("URI parser error: %w", err)
	}

	feeds, err := rssfinder.Find(rawurl)
	if err != nil {
		return err
	}

	if len(feeds) == 0 {
		return fmt.Errorf("can't detect any feeds.")
	}

	if len(feeds) == 1 {
		s.cfg.URLs[feeds[0].Href] = feeds[0].Title
	} else {
		urlWithTitle := s.ask(&feeds)
		for href, title := range urlWithTitle {
			s.cfg.URLs[href] = title
		}
	}

	return configure.Save(s.app, s.cfg)
}

func (s *Subscriber) ask(feeds *[]*rssfinder.Feed) map[string]string {
	selected := []string{}
	options := []string{}
	urlWithTitle := map[string]string{}
	dict := map[string]*rssfinder.Feed{}

	for _, feed := range *feeds {
		key := "<" + feed.Title + "> " + feed.Href
		options = append(options, key)
		dict[key] = feed
	}

	prompt := &survey.MultiSelect{
		Message: "What feeds do you want to subscribe to:",
		Options: options,
	}
	survey.AskOne(prompt, &selected)

	for _, u := range selected {
		feed := dict[u]
		urlWithTitle[feed.Href] = feed.Title
	}

	return urlWithTitle
}
