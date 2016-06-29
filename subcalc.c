/*-
 * Copyright (c) Christian S.J. Peron (csjp@sqrt.ca) 
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 * 3. The names of the authors may not be used to endorse or promote
 *    products derived from this software without specific prior written
 *    permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE AUTHOR AND CONTRIBUTORS ``AS IS'' AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED.  IN NO EVENT SHALL THE AUTHOR OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
 * OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 * HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
 * LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
 * OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
 * SUCH DAMAGE.
 *
 */
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#include <unistd.h>
#include <assert.h>
#include <stdlib.h>
#include <stdio.h>
#include <math.h>
#include <string.h>
#include <err.h>

#include "subcalc.h"

static int dorange = 0;
static char *prog = "subcalc";

static int usage(void);
static int packadrinfo(int af, u_char *adrspace, const char *str);
static char *getipaddress(int af, u_char *adrspace);
static int pl2m[9] = { 0x00, 0x80, 0xc0, 0xe0, 0xf0, 0xf8, 0xfc, 0xfe, 0xff };
static char *invert_mask(int, void *, void *);

static int
mask_discover(char *number, int af)
{
	double lg, lg2, pwr;
	u_int i, sp;
	int mx;

	sp = atoi(number);
	if (sp == 0) {
		(void) fprintf(stderr, "invalid mask specification: %s\n",
		    number);
		exit(1);
	}
	mx = (af == AF_INET6) ? 128 : 32;
	for (i = 0; i <= mx; i++) {
		pwr = pow(2, i);
		if (pwr >= sp)
			break;
	}
	lg = log((double)sp) / log(2);
	lg2 = log(pwr) / log(2);
	printf("theoretical len: %.30f\n", (double)(mx - lg));
	printf("working len:     %.0f\n", (double)(mx - lg2));
	printf("hosts:           %.0f\n", pow(2, lg2));
	return (0);
}

struct in6_addr *
plen2mask(int n)
{
	static struct in6_addr ia;
	u_char *p;
	int i;

	memset(&ia, 0, sizeof(struct in6_addr));
	p = (u_char *)&ia;
	for (i = 0; i < 16; i++, p++, n -= 8) {
		if (n >= 8) {
			*p = 0xff;
			continue;
		}
		*p = pl2m[n];
		break;
	}
	return (&ia);
}

static int
setaddrmask(struct in6_addr *ip, struct in6_addr *mask, unsigned bits)
{
	int i;

	*mask = *(plen2mask(bits));
	for(i = 0; i < sizeof(*ip); i++)
		ip->s6_addr[i] &= mask->s6_addr[i];
	return (0);
}

static int	extractbits(int af, u_char *adrspace);

struct cmdargs {
	int af;
	char address[64];
	u_long bits;
};

static int
proccmdargs(int c, char *a[], struct cmdargs *p)
{
	static char **fp, *fields[10];
	struct in_addr tmp_mask;
	struct in6_addr a6;
	char *m, *r, *tmp;
	int g;
	union {
		struct in_addr mask;
		struct in6_addr mask6;
	} msk;
	union {
		u_int32_t a;
		u_char octet[4];
	} stf;

	if (strcmp(a[1], "stf") == 0) {
		if (&a[2][0] == NULL)
			errx(1, "must specify an address family");
		else if (strcmp(a[2], "inet") == 0) {
			if (&a[3][0] == 0)
				errx(1, "must specify ip address");
			packadrinfo(AF_INET, (u_char *)&stf.a, a[3]);
		} else if (strcmp(a[2], "inet6") == 0) {
			if (&a[3][0] == 0)
				errx(1, "must specify ip6 address or net");
			sscanf(a[3], "2002:%02x%02x:%02x%02x:",
			    (unsigned int *)&stf.octet[0], (unsigned int *)&stf.octet[1],
			    (unsigned int *)&stf.octet[2], (unsigned int *)&stf.octet[3]);
		} else
			errx(1, "invalid address family: %s", a[2]);
		printf("6to4 network:        2002:%02x%02x:%02x%02x::/48\n",
		    stf.octet[0], stf.octet[1], stf.octet[2], stf.octet[3]);
		printf("ip version 4 parent: %s\n",
		    getipaddress(AF_INET, (u_char *)&stf.a));
		exit(0);
	}
	g = c - 1;
	if (strcmp(a[1], "int6") == 0) {
		if (&a[2][0] == NULL)
			errx(1, "must specify ip6 address.");
		if (&a[3][0] == NULL)
			errx(1, "must specify a hostname.");
		packadrinfo(AF_INET6, (u_char *)&a6, a[2]);
		for (g = 15; g >= 0; g--)
			printf("%x.%x.%s",
			    (a6.s6_addr8[g] & 0x0f),
			    (a6.s6_addr8[g] & 0xf0) >> 4,
			    (g == 0) ? "ip6.int.\tIN\tPTR\t" : "");
		printf("%s\n", a[3]);
		exit(0);
	}
	if (strcmp(a[1], "arpa6") == 0) {
		if (&a[2][0] == NULL)
			errx(1, "must specify ip6 address.");
		if (&a[3][0] == NULL)
			errx(1, "must specify a hostname.");
		packadrinfo(AF_INET6, (u_char *)&a6, a[2]);
		for (g = 15; g >= 0; g--)
			printf("%x.%x.%s",
			    (a6.s6_addr8[g] & 0x0f),
			    (a6.s6_addr8[g] & 0xf0) >> 4,
			    (g == 0) ? "ip6.arpa.\tIN\tPTR\t" : "");
		printf("%s\n", a[3]);
		exit(0);
	}
	if (strcmp(a[g], "print") == 0)
		dorange++;
	if (strcmp("inet", a[1]) == 0) {
		if (c < 3)
			usage();
		p->af = AF_INET;
		if (strcmp(a[2], "hosts") == 0) {
			if (&a[3][0] == NULL)
				usage();
			mask_discover(a[3], AF_INET);
			exit(0);
		}
		if (strchr(a[2], '/')) {
			tmp = &a[2][0];
			for (fp = fields; (*fp = strsep(&tmp, "/"))
				!= NULL;) {
				if (**fp != '\0')
					if (++fp >= &fields[2])
						break;
			}
			if (fields[0] == NULL || fields[1] == NULL) {
				(void) fprintf(stderr, "invalid address/cidr specification\n");
				exit(1);
			}
			memcpy(p->address, fields[0], sizeof(p->address));
			p->bits = atoi(fields[1]);
			return (0);
		} else if (c == 3)
			errx(1, "specify network bits or mask.");
		memcpy(p->address, a[2], sizeof(p->address));
		if (strcmp(a[3], "netmask") == 0) {
			if (c != 5 && !dorange)
				errx(1,"invalid words near netmask");
			m = &a[4][0];
			if (*m == '0' && *m + 1 == 'x')
				msk.mask.s_addr = strtoul(m, &r, 16);
			else
				msk.mask.s_addr = inet_addr(a[4]);
			p->bits = extractbits(AF_INET, (u_char *)&msk.mask);
		}
		if (strcmp(a[3], "mask") == 0) {
			if (c != 5 && !dorange)
				errx(1,"invalid words near mask");
			tmp_mask.s_addr = inet_addr(a[4]);
			(void) invert_mask(AF_INET, &tmp_mask,
			    &msk.mask.s_addr);
			p->bits = extractbits(AF_INET, (u_char *)&msk.mask);
		}
		if (strcmp(a[3], "prefixlen") == 0) {
			if (c != 5 && !dorange)
				errx(1,"invalid words near prefixlen");
			p->bits = atoi(a[4]);
		}
		return (0);
	}

	if (strcmp("inet6", a[1]) == 0) {
		if (c < 3)
			usage();
		p->af = AF_INET6;
		if (strcmp(a[2], "hosts") == 0) {
			if (&a[3][0] == NULL)
				usage();
			mask_discover(a[3], AF_INET6);
			exit(0);
		}
		if (strchr(a[2], '/')) {
			tmp = &a[2][0];
			for(fp = fields; (*fp = strsep(&tmp, "/"))
				!= NULL;) {
				if (**fp != '\0')
					if (++fp >= &fields[2])
						break;
			}
			if (fields[0] == NULL || fields[1] == NULL) {
				(void) fprintf(stderr, "invalid address/cidr specification\n");
				exit(1);
			}
			memcpy(p->address, fields[0], sizeof(p->address));
			p->bits = atoi(fields[1]);
			return (0);
		} else if (c == 3)
			errx(1, "specify network bits or mask.");
		memcpy(p->address, a[2], sizeof(p->address));
		if (strcmp(a[3], "prefixlen") == 0) {
			if (c != 5 && !dorange)
				errx(1,"invalid words near prefixlen");
			p->bits = atoi(a[4]);
		}
		if (strcmp(a[3], "netmask") == 0) {
			if (c != 5 && !dorange)
				errx(1,"invalid words near netmask");
			inet_pton(AF_INET6, a[4], (u_char *)&msk.mask6);
			p->bits = extractbits(AF_INET6, (u_char *)&msk.mask6);
		}
		return (0);
	}
	errx(1, "`%s' is an invalid address family.", a[1]);
	return (0);
}

static int
getb(u_char *field, unsigned pos)
{
	u_char mask;
	int i;

	mask = 0x80;
	for(i = 0; i < (pos % 8); i++)
		mask = (mask >> 1);
	return (((mask & field[(int)(pos / 8)]) == mask) ? 1 : 0);
}

static int
setb(u_char *field, unsigned pos, char state)
{
	u_char mask;
	int i;

	mask = 0x80;
	for(i = 0; i < (pos % 8); i++)
		 mask = (mask >> 1);
	if (state)
		field[pos / 8] |= mask;
	else
		field[pos / 8] &= ~mask;
	return (0);
}

static char *
invert_mask(int af, void *addr, void *conv)
{
	char buf[128];
	int i, bit;
	union {
		struct in6_addr *in6;
		struct in_addr *in;
	} adu;

	bzero(&buf[0], sizeof(buf));
	/*
	 * NB: This function was created to effectively invert the subnet mask
	 * to be used with Cisco's ACL specifications.  Last time I looked at
	 * the Cisco configuration for ACLs they used CIDR and not the
	 * masking method.
	 */
	assert(af == PF_INET);
	adu.in = (struct in_addr *)addr;
	for (i = IPWIDTH - 1; i >= 0; i--) {
		bit = getb((u_char *)&adu.in->s_addr, i);
		if (bit == 0)
			setb((u_char *)&adu.in->s_addr, i, 1);
		else
			setb((u_char *)&adu.in->s_addr, i, 0);
	}
	if (conv != NULL)
		(void) bcopy(&adu.in->s_addr, conv, sizeof(adu.in->s_addr));
	inet_ntop(af, &adu.in->s_addr, &buf[0], INET_ADDRSTRLEN);
	return (strdup(&buf[0]));
}

static int
unsetmask(int af, u_char *adrspace, unsigned b)
{
	int i;
	union {
		struct in6_addr *in6;
		struct in_addr *in;
	} adu;

	if (af == AF_INET6) {
		adu.in6 = (struct in6_addr *)adrspace;
		for(i = IPV6WIDTH-1; i >= (IPV6WIDTH - b); i--)
			setb((u_char *)&adu.in6->s6_addr8[0], i, 0);
	}
	if (af == AF_INET) {
		adu.in = (struct in_addr *)adrspace;
		for (i = IPWIDTH-1; i >= (IPWIDTH - b); i--)
			setb((u_char *)&adu.in->s_addr, i, 0);
	}
	return (0);
}

static int
extractbits(int af, u_char *adrspace)
{
	unsigned bits;
	int i;

	bits = 0;
	if (af == AF_INET6) {
		for(i = 0; i < IPV6WIDTH; i++)
			if (!getb((u_char *)adrspace, i))
				bits++;
		return (IPV6WIDTH - bits);
	}
	if (af == AF_INET) {
		for(i = 0; i < IPWIDTH; i++)
			if (!getb((u_char *)adrspace, i))
				bits++;
		return (IPWIDTH - bits);
	}
	return (-1);
}

static int
packadrinfo(int af, u_char *adrspace, const char *str)
{
	union {
		struct in6_addr *in6;
		struct in_addr *in;
	} adu;

	if (af == AF_INET6) {
		adu.in6 = (struct in6_addr *)adrspace;
		inet_pton(AF_INET6, str, adu.in6);
	}
	if (af == AF_INET) {
		adu.in = (struct in_addr *)adrspace;
		adu.in->s_addr = inet_addr(str);
		if (adu.in->s_addr == INADDR_NONE)
			errx(1, "invalid address specification");
	}
	return (0);
}

static int
setmask(int af, u_char *adrspace, unsigned b)
{
	int i;
	union {
		struct in6_addr *in6;
		struct in_addr *in;
	} adu;

	if (af == AF_INET6) {
		adu.in6 = (struct in6_addr *)adrspace;
		for (i = IPV6WIDTH-1; i >= (IPV6WIDTH - b); i--)
			setb((u_char *)&adu.in6->s6_addr8[0], i, 1);
	}
	if (af == AF_INET) {
		adu.in = (struct in_addr *)adrspace;
		for (i = IPWIDTH-1; i >= (IPWIDTH - b); i--)
			setb((u_char *)&adu.in->s_addr, i, 1);
	}
	return (0);
}

static char *
getipaddress(int af, u_char *adrspace)
{
	static char buf[64];

	bzero(&buf[0], sizeof(buf));
	inet_ntop(af, adrspace, buf, sizeof(buf));
	return (&buf[0]);
}

static int
usage(void)
{

	(void) fprintf(stderr,
	    "usage: %s [inet | inet6] address [netmask | mask ] mask <print>\n"
	    "       %s [inet | inet6] address [prefixlen] len <print>\n"
	    "       %s [inet | inet6] hosts value\n"
	    "       %s [int6 | arpa6] address hostname\n"
	    "       %s stf [inet | inet6 ] address\n",
	    prog, prog, prog, prog, prog);
	exit(1);
	/* NOT REACHED */
}

int
main(int argc, char *argv [])
{
	struct in6_addr adr6, adr62, ip6, ip6mask;
	u_int destmask, valmask;
	struct in_addr adr, adr2;
	char buf[64], *cmask;
	struct cmdargs cd;
	u_char *aaa;
	double p;
	int b, x;

	if (argc == 1)
		usage();
	proccmdargs(argc, argv, &cd);
	if (cd.af == AF_INET6) {
		SETADR6(adr6);
		SETADR6(adr62);
		SETADR6(ip6);
		setaddrmask(&ip6, &ip6mask, cd.bits);
		b = IPV6WIDTH - cd.bits;
		unsetmask(AF_INET6, (u_char *)&adr6, b);
		printf("%srange:       %s > ",
		    (dorange ? "; " : ""), getipaddress(AF_INET6,
		    (u_char *)&adr6));
		setmask(AF_INET6, (u_char *)&adr62, b);
		printf("%s\n", getipaddress(AF_INET6, (u_char *)&adr62));
		p = pow(2, (double)b);
		printf("%shosts:       %.0f\n",
		    (dorange ? "; " : ""), p);
		printf("%sprefixlen:   %lu\n",
		    (dorange ? "; " : ""), cd.bits);
		printf("%smask:        %s\n",
		    (dorange ? "; " : ""), inet_ntop(AF_INET6, &ip6mask, buf,
		    sizeof(buf)));
		if (dorange == 0)
			return (0);
		for(;;) {
			x = 15;
			if (MASKEQUAL(&adr6, &ip6mask, &ip6))
				printf("%s\n", getipaddress(AF_INET6, 
				    (u_char *)&adr6));
			else
				break;
			while (x >= 0 && (++adr6.s6_addr8[x] & 0xff) == 0)
				x--;
		}
	}
	if (cd.af == AF_INET) {
		packadrinfo(AF_INET, (u_char *)&adr, cd.address);
		packadrinfo(AF_INET, (u_char *)&adr2, cd.address);
		b = IPWIDTH - cd.bits;
		unsetmask(AF_INET, (u_char *)&adr, b);
		printf("%srange:       %s > ",
		   (dorange ? "; " : ""),
		    getipaddress(AF_INET, (u_char *)&adr));
		setmask(AF_INET, (u_char *)&adr2, b);
		printf("%s\n", getipaddress(AF_INET, (u_char *)&adr2));
		printf("%srange b10:   %u > %u\n",
		    (dorange ? "; " : ""), htonl(adr.s_addr),
		    htonl(adr2.s_addr));
		printf("%srange b16:   0x%x > 0x%x\n",
		    (dorange ? "; " : ""), htonl(adr.s_addr),
		    htonl(adr2.s_addr));
		adr2.s_addr = htonl(~0 << (32 - cd.bits));
		p = pow(2, (double)b);
		printf("%shosts:       %.0f\n",
		    (dorange ? "; " : ""), p);
		printf("%sprefixlen:   %lu\n",
		    (dorange ? "; " : ""), cd.bits);
		printf("%snetmask:     %s\n",
		    (dorange ? "; " : ""), getipaddress(AF_INET,
		    (u_char *)&adr2));
		cmask = invert_mask(AF_INET, &adr2, NULL);
		printf("%smask:        %s\n",
		    (dorange ? "; " : ""), cmask);
		if (dorange == 0)
			return (0);
		destmask = 1 << b;
		valmask = 0;
		while (valmask != destmask) {
			x = 3;
			aaa = (u_char *)&adr.s_addr;
			printf("%s\n", getipaddress(AF_INET, 
				(u_char *)&adr));
			while (x >= 0 && ((++aaa[x] & 0xff) == 0))
				x--;
			valmask++;
		}
	}
	return (0);
}
