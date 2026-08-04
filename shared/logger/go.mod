module github.com/kelrob/shared/logger

go 1.26.5

require (
	github.com/kelrob/shared/env v0.0.0-00010101000000-000000000000
	github.com/rs/zerolog v1.35.1
)

require (
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.29.0 // indirect
)

replace github.com/kelrob/shared/env => ../env
