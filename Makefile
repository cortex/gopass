default: build

OS := $(shell uname)
XDG_APPLICATION_PATH = ~/.local/share/applications
XDG_ICON_PATH = ~/.local/share/icons/hicolor/scalable/apps

build:
	go generate
	go build

install: build
	go install
ifeq ($(OS),Linux)
		mkdir -p $(XDG_APPLICATION_PATH)
		cp gopass.desktop $(XDG_APPLICATION_PATH)
		mkdir -p $(XDG_ICON_PATH)
		cp assets/logo.svg $(XDG_ICON_PATH)/gopass.svg
endif

setup:
	git submodule init
	git submodule update
	
clean:
	rm qrc.go
	go clean
