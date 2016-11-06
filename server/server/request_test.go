// +build all_tests unit_tests

package server

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"dmitryfrank.com/geekmarks/server/storage"
	//"github.com/juju/errors"
)

func TestFromWebSocketRequest(t *testing.T) {
	reader := bytes.NewReader([]byte(`
	{
		"method": "GET",
		"path": "/one/two",
		"values": {
			"name1": "single",
			"name2": ["first", "second"],
			"name3": 20,
			"name4": 20.2,
			"name5": ["first", 3, 7.3]
		},
		"body": {
			"p1": "v1",
			"p2": "v2",
			"p3": [1, 2, 3],
			"p4": {
				"p5": 5
			}
		}
	}
	`))
	wsr, err := parseWebSocketRequest(reader)
	if err != nil {
		t.Errorf("parsing error: %s", err)
	}

	caller := &storage.UserData{}
	subjUser := &storage.UserData{}

	gmr, err := makeGMRequestFromWebSocketRequest(wsr, caller, subjUser)
	if err != nil {
		t.Errorf("error making GMRequest from WebSocketRequest: %s", err)
	}

	if gmr.Method != "GET" {
		t.Errorf("expected method GET, got %q", gmr.Method)
	}

	var ok bool
	var sl []string
	sl, ok = gmr.Values["name1"]
	if !ok {
		t.Errorf("no value %q", "name1")
	}
	if len(sl) != 1 {
		t.Errorf("len of %q should be %d, but it's %d", "name1", 1, len(sl))
	}
	if sl[0] != "single" {
		t.Errorf("value[%q][%d] should be %q, but it's %q", "name1", 0, "single", sl[0])
	}

	sl, ok = gmr.Values["name2"]
	if !ok {
		t.Errorf("no value %q", "name2")
	}
	if len(sl) != 2 {
		t.Errorf("len of %q should be %d, but it's %d", "name2", 2, len(sl))
	}
	if sl[0] != "first" {
		t.Errorf("value[%q][%d] should be %q, but it's %q", "name2", 0, "first", sl[0])
	}
	if sl[1] != "second" {
		t.Errorf("value[%q][%d] should be %q, but it's %q", "name2", 1, "second", sl[1])
	}

	sl, ok = gmr.Values["name3"]
	if !ok {
		t.Errorf("no value %q", "name2")
	}
	if len(sl) != 1 {
		t.Errorf("len of %q should be %d, but it's %d", "name3", 1, len(sl))
	}
	if sl[0] != "20" {
		t.Errorf("value[%q][%d] should be %q, but it's %q", "name3", 0, "20", sl[0])
	}

}

func shouldFail(t *testing.T, str string) {
	reader := bytes.NewReader([]byte(str))
	wsr, err := parseWebSocketRequest(reader)
	if err != nil {
		t.Errorf("parsing error: %s", err)
	}

	caller := &storage.UserData{}
	subjUser := &storage.UserData{}

	_, err = makeGMRequestFromWebSocketRequest(wsr, caller, subjUser)
	if err == nil {
		t.Errorf("should not be able to convert %s", str)
	}
}

func TestFromWebSocketRequestWrong(t *testing.T) {
	shouldFail(t, `
	{
		"method": "GET",
		"path": "/one/two",
		"values": {
			"name1": "single",
			"name2": ["first", []]
		}
	}
	`)

	shouldFail(t, `
	{
		"method": "GET",
		"path": "/one/two",
		"values": {
			"name1": [{}]
		}
	}
	`)

	shouldFail(t, `
	{
		"method": "GET",
		"path": "/one/two",
		"values": {
			"name1": {}
		}
	}
	`)
}

func TestFromHttpRequest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		gmr, err := makeGMRequestFromHttpRequest(r, func(r *http.Request) (*storage.UserData, error) {
			return &storage.UserData{}, nil
		})
		if err != nil {
			t.Errorf("%s", err)
		}

		d, err := ioutil.ReadAll(gmr.Body)
		if err != nil {
			t.Errorf("%s", err)
		}

		if string(d) != "one two three" {
			t.Errorf("expected body: %q, got: %q", "one two three", string(d))
		}

		// TODO: test more fields of gmr
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	_, err := http.Post(ts.URL+"/test", "", bytes.NewReader([]byte("one two three")))
	if err != nil {
		t.Errorf("%s", err)
	}
}
