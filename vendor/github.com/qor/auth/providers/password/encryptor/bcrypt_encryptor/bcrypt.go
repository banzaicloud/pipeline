package bcrypt_encryptor

import "golang.org/x/crypto/bcrypt"

// Config BcryptEncryptor config
type Config struct {
	Cost int
}

// BcryptEncryptor BcryptEncryptor struct
type BcryptEncryptor struct {
	Config *Config
}

// New initalize BcryptEncryptor
func New(config *Config) *BcryptEncryptor {
	if config == nil {
		config = &Config{}
	}

	if config.Cost == 0 {
		config.Cost = bcrypt.DefaultCost
	}

	return &BcryptEncryptor{
		Config: config,
	}
}

// Digest generate encrypted password
func (bcryptEncryptor *BcryptEncryptor) Digest(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashedPassword), err
}

// Compare check hashed password
func (bcryptEncryptor *BcryptEncryptor) Compare(hashedPassword string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
