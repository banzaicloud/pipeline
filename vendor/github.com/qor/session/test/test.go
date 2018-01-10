package test

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/qor/session"
)

var Server *httptest.Server

type Site struct {
	SessionManager session.ManagerInterface
}

func (site Site) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/get":
		value := site.SessionManager.Get(req, req.URL.Query().Get("key"))
		w.Write([]byte(value))
	case "/pop":
		value := site.SessionManager.Pop(w, req, req.URL.Query().Get("key"))
		w.Write([]byte(value))
	case "/setflash":
		site.SessionManager.Flash(w, req, session.Message{Message: template.HTML(req.URL.Query().Get("message"))})
	case "/getflash":
		messages := []string{}
		for _, flash := range site.SessionManager.Flashes(w, req) {
			messages = append(messages, string(flash.Message))
		}
		w.Write([]byte(strings.Join(messages, ", ")))
	case "/set":
		err := site.SessionManager.Add(w, req, req.URL.Query().Get("key"), req.URL.Query().Get("value"))
		if err != nil {
			panic(fmt.Sprintf("No error should happe when set session, but got %v", err))
		}

		value := site.SessionManager.Get(req, req.URL.Query().Get("key"))
		w.Write([]byte(value))
	}
}

type server struct {
	req *http.Request
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.req = r
}

func TestAll(manager session.ManagerInterface, t *testing.T) {
	Server = httptest.NewServer(manager.Middleware(Site{SessionManager: manager}))
	newReq := func() *http.Request {
		var (
			req, _ = http.NewRequest("GET", "/", nil)
			w      = httptest.NewRecorder()
		)

		s := &server{}
		manager.Middleware(s).ServeHTTP(w, req)
		return s.req
	}

	TestWithRequest(manager, t)
	TestAddAndGet(httptest.NewRecorder(), newReq(), manager, t)
	TestAddAndPop(httptest.NewRecorder(), newReq(), manager, t)
	TestFlash(httptest.NewRecorder(), newReq(), manager, t)
	TestLoad(httptest.NewRecorder(), newReq(), manager, t)
}

func newClient(resp *http.Response) *http.Client {
	// Get cookie in another request
	cookieJar, _ := cookiejar.New(nil)
	u, _ := url.Parse(Server.URL)
	cookieJar.SetCookies(u, resp.Cookies())

	return &http.Client{Jar: cookieJar}
}

func TestWithRequest(manager session.ManagerInterface, t *testing.T) {
	maps := map[string]interface{}{
		"key":               "value",
		"中文测试":              "中文测试",
		"<html> &tag, test": "<html> &tag, test",
	}

	for key, value := range maps {
		setQuery := url.Values{}
		setQuery.Add("key", key)
		setQuery.Add("value", fmt.Sprint(value))

		// Set cookie
		resp, err := http.Get(Server.URL + "/set?" + setQuery.Encode())
		if err != nil {
			t.Errorf("no error should happen when request set cookie")
		}

		// Test get cookie in same request
		responseData, _ := ioutil.ReadAll(resp.Body)
		if string(responseData) != value {
			t.Errorf("failed to get saved session, expect %v, but got %v", value, string(responseData))
		}

		client := newClient(resp)
		getQuery := url.Values{}
		getQuery.Add("key", key)
		resp, err = client.Get(Server.URL + "/get?" + getQuery.Encode())
		if err != nil {
			t.Errorf("no error should happend when request get cookie")
		}

		responseData2, _ := ioutil.ReadAll(resp.Body)
		if string(responseData2) != value {
			t.Errorf("failed to get saved session, expect %v, but got %v", value, string(responseData2))
		}

		resp, err = client.Get(Server.URL + "/pop?" + getQuery.Encode())
		if err != nil {
			t.Errorf("no error should happend when request pop cookie")
		}

		responseData3, _ := ioutil.ReadAll(resp.Body)
		if string(responseData3) != value {
			t.Errorf("failed to pop saved session, expect %v, but got %v", value, string(responseData3))
		}

		resp, err = client.Get(Server.URL + "/get?" + getQuery.Encode())
		if err != nil {
			t.Errorf("no error should happend when request pop cookie")
		}

		responseData4, _ := ioutil.ReadAll(resp.Body)
		if string(responseData4) != "" {
			t.Errorf("should not be able to get session data after pop, but got %v", string(responseData4))
		}

		_, err = client.Get(Server.URL + "/setflash?message=message1")
		if err != nil {
			t.Errorf("no error should happend when request set flash")
		}

		_, err = client.Get(Server.URL + "/setflash?message=message2")
		if err != nil {
			t.Errorf("no error should happend when request set flash")
		}

		resp, err = client.Get(Server.URL + "/getflash")
		if err != nil {
			t.Errorf("no error should happend when request get flash")
		}

		responseData5, _ := ioutil.ReadAll(resp.Body)
		if string(responseData5) != "message1, message2" {
			t.Errorf("should be able to get saved flash data, but got %v", string(responseData5))
		}

		resp, err = client.Get(Server.URL + "/getflash")
		if err != nil {
			t.Errorf("no error should happend when request get flash")
		}

		responseData6, _ := ioutil.ReadAll(resp.Body)
		if string(responseData6) != "" {
			t.Errorf("should get blank string when get flashes second time, but got %v", string(responseData6))
		}
	}
}

func TestAddAndGet(w http.ResponseWriter, req *http.Request, manager session.ManagerInterface, t *testing.T) {
	if err := manager.Add(w, req, "key", "value"); err != nil {
		t.Errorf("Should add session correctly, but got %v", err)
	}

	if value := manager.Get(req, "key"); value != "value" {
		t.Errorf("failed to fetch saved session value, got %#v", value)
	}

	if value := manager.Get(req, "key"); value != "value" {
		t.Errorf("possible to re-fetch saved session value, got %#v", value)
	}
}

func TestAddAndPop(w http.ResponseWriter, req *http.Request, manager session.ManagerInterface, t *testing.T) {
	if err := manager.Add(w, req, "key", "value"); err != nil {
		t.Errorf("Should add session correctly, but got %v", err)
	}

	if value := manager.Pop(w, req, "key"); value != "value" {
		t.Errorf("failed to fetch saved session value, got %#v", value)
	}

	if value := manager.Pop(w, req, "key"); value == "value" {
		t.Errorf("can't re-fetch saved session value after get with Pop, got %#v", value)
	}
}

func TestFlash(w http.ResponseWriter, req *http.Request, manager session.ManagerInterface, t *testing.T) {
	if err := manager.Flash(w, req, session.Message{
		Message: "hello1",
	}); err != nil {
		t.Errorf("No error should happen when add Flash, but got %v", err)
	}

	if err := manager.Flash(w, req, session.Message{
		Message: "hello2",
	}); err != nil {
		t.Errorf("No error should happen when add Flash, but got %v", err)
	}

	flashes := manager.Flashes(w, req)
	if len(flashes) != 2 {
		t.Errorf("should find 2 flash messages")
	}

	flashes2 := manager.Flashes(w, req)
	if len(flashes2) != 0 {
		t.Errorf("flash should be cleared when fetch it second time, but got %v", len(flashes2))
	}
}

func TestLoad(w http.ResponseWriter, req *http.Request, manager session.ManagerInterface, t *testing.T) {
	type result struct {
		Name    string
		Age     int
		Actived bool
	}

	user := result{Name: "jinzhu", Age: 18, Actived: true}
	manager.Add(w, req, "current_user", user)

	var user1 result
	if err := manager.Load(req, "current_user", &user1); err != nil {
		t.Errorf("no error should happen when Load struct")
	}

	if user1.Name != user.Name || user1.Age != user.Age || user1.Actived != user.Actived {
		t.Errorf("Should be able to add, load struct, ")
	}

	var user2 result
	if err := manager.Load(req, "current_user", &user2); err != nil {
		t.Errorf("no error should happen when Load struct")
	}

	if user2.Name != user.Name || user2.Age != user.Age || user2.Actived != user.Actived {
		t.Errorf("Should be able to load struct more than once")
	}

	var user3 result
	if err := manager.PopLoad(w, req, "current_user", &user3); err != nil {
		t.Errorf("no error should happen when PopLoad struct")
	}

	if user3.Name != user.Name || user3.Age != user.Age || user3.Actived != user.Actived {
		t.Errorf("Should be able to add, pop load struct")
	}

	var user4 result
	if err := manager.Load(req, "current_user", &user4); err != nil {
		t.Errorf("Should return error when fetch data after PopLoad")
	}
}
