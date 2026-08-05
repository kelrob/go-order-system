module github.com/kelrob/shared/middleware

go 1.26.5

replace github.com/kelrob/shared/response => ../response

replace github.com/kelrob/shared/ulid => ../ulid

require github.com/kelrob/shared/response v0.0.0-00010101000000-000000000000

require github.com/golang-jwt/jwt v3.2.2+incompatible

require (
	github.com/kelrob/shared/ulid v0.0.0-00010101000000-000000000000 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
)
