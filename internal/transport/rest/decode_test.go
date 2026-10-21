package rest

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRejectsTrailingContent(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "second JSON value", body: `{"name":"test"} {"name":"extra"}`},
		{name: "invalid trailing text", body: `{"name":"test"} trailing`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			var destination struct {
				Name string `json:"name"`
			}
			if err := decodeJSON(rec, req, &destination); err == nil {
				t.Fatal("expected an error for trailing content")
			}
		})
	}
}

func TestDecodeJSONRejectsOversizedBody(t *testing.T) {
	body := `{"name":"` + strings.Repeat("a", (1<<20)+1) + `"}`

	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	rec := httptest.NewRecorder()

	var destination struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(rec, req, &destination); err == nil {
		t.Fatal("expected an error for oversized JSON body")
	}
}
