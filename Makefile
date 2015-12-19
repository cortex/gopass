default: all

all:
	go get gopkg.in/qml.v1/cmd/genqrc
	go generate
	go build

.PHONY: install
install:
	go install

.PHONY: clean
clean:
	rm qrc.go
	go clean
