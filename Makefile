TARGET=evilginx
PACKAGES=core database log parser

.PHONY: all build clean
all: build

build:
	@go build -o ./bin/$(TARGET) -mod=vendor main.go

clean:
	@go clean
	@rm -f ./bin/$(TARGET)

install:
	@mkdir -p /usr/share/evilginx/phishlets
	@cp ./phishlets/* /usr/share/evilginx/phishlets/
	@cp ./bin/$(TARGET) /usr/local/bin
