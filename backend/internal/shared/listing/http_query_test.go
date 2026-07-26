package listing

import (
	"net/http/httptest"
	"testing"
)

func TestParseHTTPQuery(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest("GET", "/items?search=backend&page=2&pageSize=10", nil)
	query, err := ParseHTTPQuery(request)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if query != (Query{Search: "backend", Page: 2, PageSize: 10}) {
		t.Fatalf("unexpected query: %#v", query)
	}
}

func TestParseHTTPQueryDefaults(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest("GET", "/items", nil)
	query, err := ParseHTTPQuery(request)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if query != (Query{Page: DefaultPage, PageSize: DefaultPageSize}) {
		t.Fatalf("unexpected defaults: %#v", query)
	}
}

func TestParseHTTPQueryRejectsInvalidPagination(t *testing.T) {
	t.Parallel()

	for _, target := range []string{"/items?page=0", "/items?page=invalid", "/items?pageSize=0", "/items?pageSize=101"} {
		request := httptest.NewRequest("GET", target, nil)
		if _, err := ParseHTTPQuery(request); err == nil {
			t.Fatalf("expected %q to be rejected", target)
		}
	}
}
