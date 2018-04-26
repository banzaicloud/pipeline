package render

import (
	"regexp"
	"testing"

	"net/http/httptest"
	"net/textproto"
)

func TestExecute(t *testing.T) {
	Render := New(nil, "test")

	request := httptest.NewRequest("GET", "/test", nil)
	responseWriter := httptest.NewRecorder()
	var context interface{}

	tmpl := Render.Layout("layout_for_test")
	tmpl.Execute("test", context, request, responseWriter)

	if textproto.TrimString(responseWriter.Body.String()) != "Template for test" {
		t.Errorf("The template isn't rendered")
	}
}

func TestErrorMessageWhenMissingLayout(t *testing.T) {
	Render := New(nil, "test")

	request := httptest.NewRequest("GET", "/test", nil)
	responseWriter := httptest.NewRecorder()
	var context interface{}

	not_exist_layout := "ThePlant"
	tmpl := Render.Layout(not_exist_layout)
	err := tmpl.Execute(" test", context, request, responseWriter)

	errorRegexp := "Failed to render layout:.+" + not_exist_layout + ".*"

	if matched, _ := regexp.MatchString(errorRegexp, err.Error()); !matched {
		t.Errorf("Missing layout error message is incorrect")
	}
}
