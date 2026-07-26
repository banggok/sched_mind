package listing

import (
	"errors"
	"net/http"
	"strconv"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 5
	MaxPageSize     = 100
)

// ParseHTTPQuery maps the standard list query parameters into a validated Query.
func ParseHTTPQuery(request *http.Request) (Query, error) {
	query := Query{
		Search:   request.URL.Query().Get("search"),
		Page:     DefaultPage,
		PageSize: DefaultPageSize,
	}

	var err error
	if value := request.URL.Query().Get("page"); value != "" {
		query.Page, err = strconv.Atoi(value)
		if err != nil || query.Page < 1 {
			return Query{}, errors.New("page must be a positive integer")
		}
	}
	if value := request.URL.Query().Get("pageSize"); value != "" {
		query.PageSize, err = strconv.Atoi(value)
		if err != nil || query.PageSize < 1 || query.PageSize > MaxPageSize {
			return Query{}, errors.New("pageSize must be between 1 and 100")
		}
	}

	return query, nil
}
