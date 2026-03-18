.PHONY: build clean install install-local uninstall deb

# Installation paths (can be overridden)
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
MANDIR ?= $(PREFIX)/share/man/man1

build:
	mkdir -p build
	go build -o build/lazysystemd ./cmd/lazysystemd

clean:
	rm -rf build
	rm -rf debian/lazysystemd
	rm -f *.deb *.buildinfo *.changes
	dh_clean

# Install to system directories (requires sudo)
install: build
	install -D -m 755 build/lazysystemd $(DESTDIR)$(BINDIR)/lazysystemd
	install -D -m 644 lazysystemd.1 $(DESTDIR)$(MANDIR)/lazysystemd.1
	@echo "Installed to $(DESTDIR)$(BINDIR)/lazysystemd"
	@echo "Installed man page to $(DESTDIR)$(MANDIR)/lazysystemd.1"

# Install to /usr/local (default, for local development)
install-local: PREFIX=/usr/local
install-local: install

# Uninstall (removes files installed by install-local)
uninstall:
	rm -f $(BINDIR)/lazysystemd
	rm -f $(MANDIR)/lazysystemd.1
	@echo "Uninstalled lazysystemd"

# Debian package build (uses DESTDIR)
deb:
	dpkg-buildpackage -b -uc -us
