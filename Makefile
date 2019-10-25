GO_SOURCES = audio.go avconv.go client.go go.mod go.sum \
             playable.go ./cmd/telecom-native/main.go

GO ?= go

.PHONY: clean all

all: telecom.so telecom.a telecom.h

telecom.so : $(GO_SOURCES)
	$(GO) build -o telecom.so $(GOFLAGS) -buildmode=c-shared \
                       ./cmd/telecom-native/main.go

telecom.a : $(GO_SOURCES)
	$(GO) build -o telecom.a $(GOFLAGS) -buildmode=c-archive \
                       ./cmd/telecom-native/main.go

telecom.h : telecom.so telecom.a

clean:
	rm -f telecom.so telecom.a telecom.h

ifeq ($(PREFIX),)
    PREFIX = /usr/local
endif

install: telecom.so telecom.a telecom.h
	install -d $(DESTDIR)$(PREFIX)/lib/
	install -m 644 telecom.a $(DESTDIR)$(PREFIX)/lib/libtelecom.a
	install -m 644 telecom.so $(DESTDIR)$(PREFIX)/lib/libtelecom.so
	install -d $(DESTDIR)$(PREFIX)/include/
	install -m 644 telecom.h $(DESTDIR)$(PREFIX)/include/telecom.h
