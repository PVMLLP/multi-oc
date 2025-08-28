APP=moc
VERSION?=0.1.1

.PHONY: all build linux vendor clean airgap

all: build

vendor:
	go mod vendor

build:
	GO111MODULE=on go build -mod=vendor -o $(APP) .

linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=on go build -mod=vendor -ldflags="-s -w" -o $(APP)-linux-amd64 .

airgap: vendor linux
	mkdir -p dist
	cp $(APP)-linux-amd64 dist/
	cp README.md dist/
	echo "Build: $$(date -u +%Y-%m-%dT%H:%M:%SZ)" > dist/README_AIRGAP.txt
	echo "Version: $(VERSION)" >> dist/README_AIRGAP.txt
	tar -C dist -czf $(APP)-airgap-$(VERSION).tar.gz $(APP)-linux-amd64 README.md README_AIRGAP.txt

clean:
	rm -rf vendor dist $(APP) $(APP)-linux-amd64 $(APP)-airgap-*.tar.gz


