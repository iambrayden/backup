module github.com/waylontech/backup/agent

go 1.24

require (
	github.com/waylontech/backup/pkg v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.28.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/klauspost/cpuid/v2 v2.0.12 // indirect
	github.com/zeebo/blake3 v0.2.4 // indirect
	go.uber.org/multierr v1.10.0 // indirect
)

replace github.com/waylontech/backup/pkg => ../pkg
