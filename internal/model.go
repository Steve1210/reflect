package internal

type Reflection struct {
	ReflectionHeader
	Body string `json:"body"`
}

type ReflectionHeader struct {
	Id        int64
	Title     string   `json:"title"`
	Tags      []string `json:"tags"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

// TODO: Add options to filter via created/edited timestamp
type FilterOptions struct {
	Title         string
	Tags          []string
	CreatedAfter  int64
	CreatedBefore int64
	UpdatedAfter  int64
	UpdatedBefore int64
}
