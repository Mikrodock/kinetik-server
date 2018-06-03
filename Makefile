GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean


BINARY=kinetik-server

all: build deploy
build:
	$(GOBUILD) -a -tags netgo -ldflags '-w' -o $(BINARY) main.go
clean:
	$(GOCLEAN)
	rm -rf $(BINARY)
deploy:
	scp $(BINARY) root@nsurleraux.be:/srv/http/cv/$(BINARY)
