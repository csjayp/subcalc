#
#
# $Header: /usr/cvs/subcalc/Makefile,v 1.6 2003/08/26 21:29:49 modulus Exp $
#

PREFIX?=/usr/local
CFLAGS+=	-pipe -O2 -Wall
OBJS=	subcalc.c
LIBS=	-lm
CC?=	CC

all: subcalc

subcalc:	$(OBJS)
	$(CC) -o $@ $(OBJS) $(LIBS) $(CFLAGS)

install:
	cp subcalc $(PREFIX)/bin
	cp subcalc.1.gz $(PREFIX)/man/man1/

deinstall:
	rm -f $(PREFIX)/man/man1/subcalc.1.gz
	rm -f $(PREFIX)/bin/subcalc

clean:
	rm -f subcalc
