.PHONY: build release-artifacts clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.1.0)
DIST ?= dist
SERVER_URL ?= http://127.0.0.1:1223/geolocate
EXT_VERSION ?= 1.1

build:
	mkdir -p $(DIST)
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(DIST)/ancaeus .

release-artifacts: build
	mkdir -p $(DIST)/packaging $(DIST)/chromium-extension
	cp -a packaging/. $(DIST)/packaging/
	cp chromium-extension/background.js chromium-extension/content.js \
		chromium-extension/inject.js chromium-extension/manifest.json \
		$(DIST)/chromium-extension/
	sed -i 's|__SERVER_URL__|$(SERVER_URL)|g' $(DIST)/chromium-extension/background.js $(DIST)/chromium-extension/manifest.json
	sed -i 's|__VERSION__|$(EXT_VERSION)|g' $(DIST)/chromium-extension/manifest.json
	cd $(DIST) && zip -r ancaeus-packaging.zip packaging
	cd $(DIST) && zip -r ancaeus-chromium-extension.zip chromium-extension
	cd $(DIST) && tar -czf ancaeus-linux-amd64.tar.gz ancaeus
	printf '%s\n' "$(VERSION)" > $(DIST)/VERSION

clean:
	rm -rf $(DIST)
