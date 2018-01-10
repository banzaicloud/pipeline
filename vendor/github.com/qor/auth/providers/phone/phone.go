package phone

import "github.com/qor/auth"

func New() *PhoneProvider {
	return &PhoneProvider{}
}

// PhoneProvider provide login with phone method
type PhoneProvider struct {
}

// GetName return provider name
func (PhoneProvider) GetName() string {
	return "phone"
}

// ConfigAuth config auth
func (PhoneProvider) ConfigAuth(*auth.Auth) {
}

// Login implemented login with phone provider
func (PhoneProvider) Login(context *auth.Context) {
}

// Logout implemented logout with phone provider
func (PhoneProvider) Logout(context *auth.Context) {
}

// Register implemented register with phone provider
func (PhoneProvider) Register(context *auth.Context) {
}

// Callback implement Callback with phone provider
func (PhoneProvider) Callback(*auth.Context) {
}

// ServeHTTP implement ServeHTTP with phone provider
func (PhoneProvider) ServeHTTP(*auth.Context) {
}
