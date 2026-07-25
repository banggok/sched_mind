package listing

type Query struct {
	Search   string
	Page     int
	PageSize int
}

type Page[T any] struct {
	Items    []T
	Page     int
	PageSize int
	Total    int64
}
