default: all

uname_S := $(shell sh -c 'uname -s 2>/dev/null || echo not')

all:
	go get gopkg.in/qml.v1/cmd/genqrc
	go generate
	go build

.PHONY: install
install:
	go install
ifeq ($(uname_S),Linux)
		cp gopass.desktop ~/.local/share/applications
		cp assets/logo.svg ~/.local/share/icons/hicolor/scalable/apps/gopass.svg
endif

.PHONY: clean
clean:
	rm qrc.go
	go clean
