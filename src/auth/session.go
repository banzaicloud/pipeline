// Copyright © 2020 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
)

// NewSessionManager initialize session manager based on gorilla/sessions
func NewSessionManager(sessionName string, store sessions.Store) *SessionManager {
	return &SessionManager{SessionName: sessionName, Store: store}
}

// SessionManager session manager struct for gorilla/sessions
type SessionManager struct {
	SessionName string
	Store       sessions.Store
}

const reader ContextKey = "gorilla_reader"

func (sm SessionManager) getSession(req *http.Request) (*sessions.Session, error) {
	if r, ok := req.Context().Value(reader).(*http.Request); ok {
		return sm.Store.Get(r, sm.SessionName)
	}
	return sm.Store.Get(req, sm.SessionName)
}

func (sm SessionManager) saveSession(w http.ResponseWriter, req *http.Request) {
	if session, err := sm.getSession(req); err == nil {
		if err := session.Save(req, w); err != nil {
			log.Error("no error should happen when saving session data", map[string]interface{}{"error": err})
		}
	}
}

// Add value to session data, if value is not string, will marshal it into JSON encoding and save it into session data.
func (sm SessionManager) Add(w http.ResponseWriter, req *http.Request, key string, value interface{}) error {
	defer sm.saveSession(w, req)

	session, err := sm.getSession(req)
	if err != nil {
		return err
	}

	if str, ok := value.(string); ok {
		session.Values[key] = str
	} else {
		result, _ := json.Marshal(value)
		session.Values[key] = string(result)
	}

	return nil
}

// Pop value from session data
func (sm SessionManager) Pop(w http.ResponseWriter, req *http.Request, key string) string {
	defer sm.saveSession(w, req)

	if session, err := sm.getSession(req); err == nil {
		if value, ok := session.Values[key]; ok {
			delete(session.Values, key)
			return fmt.Sprint(value)
		}
	}
	return ""
}

// Get value from session data
func (sm SessionManager) Get(req *http.Request, key string) string {
	if session, err := sm.getSession(req); err == nil {
		if value, ok := session.Values[key]; ok {
			return fmt.Sprint(value)
		}
	}
	return ""
}

// Load get value from session data and unmarshal it into result
func (sm SessionManager) Load(req *http.Request, key string, result interface{}) error {
	value := sm.Get(req, key)
	if value != "" {
		return json.Unmarshal([]byte(value), result)
	}
	return nil
}

// PopLoad pop value from session data and unmarshal it into result
func (sm SessionManager) PopLoad(w http.ResponseWriter, req *http.Request, key string, result interface{}) error {
	value := sm.Pop(w, req, key)
	if value != "" {
		return json.Unmarshal([]byte(value), result)
	}
	return nil
}

// Middleware returns a new session manager middleware instance
func (sm SessionManager) Middleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := context.WithValue(req.Context(), reader, req)
		handler.ServeHTTP(w, req.WithContext(ctx))
	})
}
