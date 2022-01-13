ci:
	@goimports -l . || (goimports -d . && exit 1)
	@golangci-lint run
	@go test -v .
.DEFAULT_GOAL := ci