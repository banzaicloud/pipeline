package beego_session_test

import (
	"encoding/json"
	"testing"

	"github.com/astaxie/beego/session"
	"github.com/qor/session/beego_session"
	"github.com/qor/session/test"
)

func TestAll(t *testing.T) {
	config := `{"cookieName":"gosessionid","enableSetCookie":true,"gclifetime":3600,"ProviderConfig":"{\"cookieName\":\"gosessionid\",\"securityKey\":\"beegocookiehashkey\"}"}`
	conf := new(session.ManagerConfig)
	if err := json.Unmarshal([]byte(config), conf); err != nil {
		t.Fatal("json decode error", err)
	}

	globalSessions, _ := session.NewManager("memory", conf)
	go globalSessions.GC()

	engine := beego_session.New(globalSessions)
	test.TestAll(engine, t)
}
