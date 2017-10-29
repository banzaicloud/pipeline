package auth0

import (
	"time"

	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

var (
	// The default generated token by Chrome jwt extension
	defaultSecret         = []byte("secret")
	defaultAudience       = []string{"audience"}
	defaultIssuer         = "issuer"
	defaultSecretProvider = NewKeyProvider(defaultSecret)
)

func getTestToken(audience []string, issuer string, expTime time.Time, alg jose.SignatureAlgorithm, key interface{}) string {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: alg, Key: key}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		panic(err)
	}

	cl := jwt.Claims{
		Issuer:   issuer,
		Audience: audience,
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		Expiry:   jwt.NewNumericDate(expTime),
	}

	raw, err := jwt.Signed(signer).Claims(cl).CompactSerialize()
	if err != nil {
		panic(err)
	}
	return raw
}
