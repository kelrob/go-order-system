module github.com/kelrob/notification-service

go 1.26.5

require (
	github.com/jackc/pgx/v5 v5.10.0
	github.com/kelrob/shared/events v0.0.0-00010101000000-000000000000
	github.com/kelrob/shared/logger v0.0.0-00010101000000-000000000000
	github.com/twmb/franz-go v1.21.5
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/kelrob/shared/helpers v0.0.0-00010101000000-000000000000 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pierrec/lz4/v4 v4.1.26 // indirect
	github.com/rs/zerolog v1.35.1 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.13.1 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)

replace github.com/kelrob/shared/logger => ../shared/logger

replace github.com/kelrob/shared/events => ../shared/events

replace github.com/kelrob/shared/helpers => ../shared/helpers
