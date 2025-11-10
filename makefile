.PHONY: run build

run:
	@go run cmd/main.go
build:
	@go build -o spider cmd/main.go && scp spider fengmengfan@121.43.115.61:/home/fengmengfan/spider_test && rm spider

