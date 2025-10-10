module github.com/mrz1836/go-pre-commit

go 1.24.0

require (
	github.com/fatih/color v1.18.0
	github.com/joho/godotenv v1.6.0-pre.2
	github.com/mattn/go-isatty v0.0.20
	github.com/spf13/cobra v1.10.1
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	golang.org/x/sys v0.37.0 // indirect
)

// Replace directive to pin godotenv to v1.5.1 due to parser panic in v1.6.0-pre.2
replace github.com/joho/godotenv v1.6.0-pre.2 => github.com/joho/godotenv v1.5.1
