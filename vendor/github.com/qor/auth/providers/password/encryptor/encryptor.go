package encryptor

// Interface encryptor interface
type Interface interface {
	Digest(password string) (string, error)
	Compare(hashedPassword string, password string) error
}
