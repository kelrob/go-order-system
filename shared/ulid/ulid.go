package ulid

import (
	"crypto/rand"

	"github.com/oklog/ulid/v2"
)

func Generate() string {
	entropy := ulid.Monotonic(rand.Reader, 0)
	return ulid.MustNew(ulid.Now(), entropy).String()
}

func Parse(ulidString string) (ulid.ULID, error) {
	ulid, err := ulid.Parse(ulidString)
	return ulid, err
}
