package yomu

type Config struct {
	URLs         map[string]string `toml:"urls"`
	Browser      string            `toml:"browser"`
	LastAccessed int64             `toml:"last_accessed"`
	Timeout      int               `toml:"timeout"`
}

type Item struct {
	Title       string
	Link        string
	Description string
}

func (i *Item) String() string {
	return i.Title + " - " + i.Link
}
