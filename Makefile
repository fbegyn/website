all: package

build:
	go build -o bin/website -ldflags "-w -s\
		-X main.Branch=$(shell git rev-parse --abbrev-ref HEAD)\
		-X main.Revision=$(shell git rev-list -1 HEAD)"\
		./cmd/server

clean:
	rm -rf bin/website
	rm -f website.{deb,rpm}

package: build
	nfpm pkg --target website.deb
	nfpm pkg --target website.rpm

.PHONY: deploy
deploy:
	@test -n "$(DEPLOY_TARGET)" -a -n "$(DEPLOY_ROOT)" || { \
		echo "usage: make deploy DEPLOY_TARGET=user@host DEPLOY_ROOT=/path"; exit 1; }
	./scripts/website-deploy $(DEPLOY_TARGET) $(DEPLOY_ROOT)
