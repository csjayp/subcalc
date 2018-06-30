#
#
PREFIX?=/usr/local
CFLAGS+=	-pipe -O2 -Wall -g
OBJS=	subcalc.c
LIBS=	-lm
CC?=	cc
TARGETS=	subcalc

all: $(TARGETS)

subcalc:	$(OBJS)
	$(CC) -o $@ $(OBJS) $(LIBS) $(CFLAGS)

install:
	[ -d $(PREFIX)/bin ] || mkdir -p $(PREFIX)/bin
	cp subcalc $(PREFIX)/bin
	[ -d $(PREFIX)/share/man/man1/ ] || mkdir -p $(PREFIX)/share/man/man1/
	cp subcalc.1 $(PREFIX)/share/man/man1/

deinstall:
	rm -f $(PREFIX)/share/man/man1/subcalc.1.gz
	rm -f $(PREFIX)/bin/subcalc

clean:
	rm -f subcalc
