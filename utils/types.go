package utils

type Config struct {
	URLs    map[string]string `toml:"urls"`
	Browser string            `toml:"browser"`
}

type Item struct {
	Title       string
	Link        string
	Description string
}

func (i *Item) String() string {
	return i.Title + " - " + i.Link
}
