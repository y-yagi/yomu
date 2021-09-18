package yomu

type Config struct {
	URLs        map[string]string `toml:"urls"`
	Browser     string            `toml:"browser"`
	LastFetched map[string]int64  `toml:"last_fetched"`
	Timeout     int               `toml:"timeout"`
}

type Item struct {
	Title       string
	Link        string
	Description string
}

func (i *Item) String() string {
	return i.Title + " - " + i.Link
}
