APP=moc
VERSION?=0.1.1

.PHONY: all build linux vendor clean airgap release

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
	@V=$(VERSION); \
	if [ -z "$$V" ]; then \
	  if ls -1 $(APP)-airgap-*.tar.gz >/dev/null 2>&1; then \
	    V=$$(ls -1 $(APP)-airgap-*.tar.gz | sed -E 's/.*-([0-9]+)\.([0-9]+)\.([0-9]+)\.tar\.gz/\1 \2 \3/' | awk 'BEGIN{a=0;b=0;c=-1}{if($$1>a||($$1==a&&($$2>b||($$2==b&&$$3>c)))){a=$$1;b=$$2;c=$$3}}END{printf("%d.%d.%d",a,b,c+1)}'); \
	  else \
	    V=0.1.0; \
	  fi; \
	fi; \
	echo "Build: $$(date -u +%Y-%m-%dT%H:%M:%SZ)" > dist/README_AIRGAP.txt; \
	echo "Version: $$V" >> dist/README_AIRGAP.txt; \
	tar -C dist -czf $(APP)-airgap-$$V.tar.gz $(APP)-linux-amd64 README.md README_AIRGAP.txt

release:
	@NV=$$( if ls -1 $(APP)-airgap-*.tar.gz >/dev/null 2>&1; then \
	  ls -1 $(APP)-airgap-*.tar.gz | sed -E 's/.*-([0-9]+)\.([0-9]+)\.([0-9]+)\.tar\.gz/\1 \2 \3/' | awk 'BEGIN{a=0;b=0;c=-1}{if($$1>a||($$1==a&&($$2>b||($$2==b&&$$3>c)))){a=$$1;b=$$2;c=$$3}}END{printf("%d.%d.%d",a,b,c+1)}'; \
	else echo 0.1.0; fi ); \
	echo "Next version: $$NV"; \
	$(MAKE) airgap VERSION=$$NV

clean:
	rm -rf vendor dist $(APP) $(APP)-linux-amd64 $(APP)-airgap-*.tar.gz


