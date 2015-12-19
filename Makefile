default: all

all:
	go generate
	go build

.PHONY: install
install:
	go install

.PHONY: clean
clean:
	rm qrc.go
	go clean
