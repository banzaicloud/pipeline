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

func (sm SessionManager) getSession(req *http.Request) (*sessions.Session, error) {
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
func (sm SessionManager) Add(w http.ResponseWriter, req *http.Request, key string, value string) error {
	defer sm.saveSession(w, req)

	session, err := sm.getSession(req)
	if err != nil {
		return err
	}

	session.Values[key] = value

	return nil
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
