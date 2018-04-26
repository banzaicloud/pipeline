package middlewares

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"
)

func registerMiddlewareRandomly(registeredMiddlewares []Middleware) *MiddlewareStack {
	stack := &MiddlewareStack{}
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	sort.Slice(registeredMiddlewares, func(i, j int) bool {
		return r.Intn(100)%2 == 1
	})

	for _, m := range registeredMiddlewares {
		stack.Use(m)
	}

	return stack
}

func registerMiddleware(registeredMiddlewares []Middleware) *MiddlewareStack {
	stack := &MiddlewareStack{}

	for _, m := range registeredMiddlewares {
		stack.Use(m)
	}

	return stack
}

func checkSortedMiddlewares(stack *MiddlewareStack, expectedNames []string, t *testing.T) {
	var (
		sortedNames          []string
		sortedMiddlewares, _ = stack.sortMiddlewares()
	)

	for _, middleware := range sortedMiddlewares {
		sortedNames = append(sortedNames, middleware.Name)
	}

	if fmt.Sprint(sortedNames) != fmt.Sprint(expectedNames) {
		t.Errorf("Expected sorted middleware is %v, but got %v", strings.Join(expectedNames, ", "), strings.Join(sortedNames, ", "))
	}
}

func TestCompileMiddlewares(t *testing.T) {
	availableMiddlewares := []Middleware{{Name: "cookie"}, {Name: "flash", InsertAfter: []string{"cookie"}}, {Name: "auth", InsertAfter: []string{"flash"}}}

	stack := registerMiddlewareRandomly(availableMiddlewares)
	checkSortedMiddlewares(stack, []string{"cookie", "flash", "auth"}, t)
}

func TestCompileComplicatedMiddlewares(t *testing.T) {
	availableMiddlewares := []Middleware{{Name: "A"}, {Name: "B", InsertBefore: []string{"C", "D"}}, {Name: "C", InsertAfter: []string{"E"}}, {Name: "D", InsertAfter: []string{"E"}, InsertBefore: []string{"C"}}, {Name: "E", InsertBefore: []string{"B"}, InsertAfter: []string{"A"}}}
	stack := registerMiddlewareRandomly(availableMiddlewares)

	checkSortedMiddlewares(stack, []string{"A", "E", "B", "D", "C"}, t)
}

func TestConflictingMiddlewares(t *testing.T) {
	t.Skipf("conflicting middlewares")
}

func TestMiddlewaresWithRequires(t *testing.T) {
	availableMiddlewares := []Middleware{{Name: "flash", Requires: []string{"cookie"}}, {Name: "session"}}
	stack := registerMiddlewareRandomly(availableMiddlewares)

	if _, err := stack.sortMiddlewares(); err == nil {
		t.Errorf("Should return error as required middleware doesn't exist")
	}
}
