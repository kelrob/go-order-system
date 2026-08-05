module github.com/kelrob/auth-service

go 1.26.5

replace github.com/kelrob/shared/logger => ../shared/logger

replace github.com/kelrob/shared/env => ../shared/env

replace github.com/kelrob/shared/password => ../shared/password

replace github.com/kelrob/shared/email => ../shared/email

replace github.com/kelrob/shared/ulid => ../shared/ulid

replace github.com/kelrob/shared/events => ../shared/events

replace github.com/kelrob/shared/middleware => ../shared/middleware

replace github.com/kelrob/shared/response => ../shared/response

replace github.com/kelrob/shared/validation => ../shared/validation

require (
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/jackc/pgx/v5 v5.10.0
	github.com/kelrob/shared/env v0.0.0-00010101000000-000000000000
	github.com/kelrob/shared/events v0.0.0-00010101000000-000000000000
	github.com/kelrob/shared/logger v0.0.0-00010101000000-000000000000
	github.com/kelrob/shared/middleware v0.0.0-00010101000000-000000000000
	github.com/kelrob/shared/password v0.0.0-00010101000000-000000000000
	github.com/kelrob/shared/response v0.0.0-00010101000000-000000000000
	github.com/kelrob/shared/ulid v0.0.0-00010101000000-000000000000
	github.com/kelrob/shared/validation v0.0.0-00010101000000-000000000000
)

require (
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/rs/zerolog v1.35.1 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
)
