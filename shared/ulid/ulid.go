package ulid

import (
	"crypto/rand"

	"github.com/oklog/ulid/v2"
)

func Generate() string {
	entropy := ulid.Monotonic(rand.Reader, 0)
	return ulid.MustNew(ulid.Now(), entropy).String()
}
