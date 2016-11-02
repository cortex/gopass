default: build

OS := $(shell uname)

build:
	git submodule init
	git submodule update
	go generate
	go build

install: build
	go install
ifeq ($(OS),Linux)
		mkdir -p ~/.local/share/applications && cp gopass.desktop ~/.local/share/applications
		mkdir -p ~/.local/share/icons/hicolor/scalable/apps && cp assets/logo.svg ~/.local/share/icons/hicolor/scalable/apps/gopass.svg
endif

clean:
	rm qrc.go
	go clean
