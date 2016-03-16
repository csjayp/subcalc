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
#ifndef SUBCALC_DOT_H_
#define SUBCALC_DOT_H_

#define	IPV6WIDTH	128
#define	IPWIDTH		32	

#ifndef LONG_MAX
#define LONG_MAX 0x7fffffffL
#endif

#if defined(__linux__)
#define s6_addr32 __in6_u.__u6_addr32
#define s6_addr8 __in6_u.__u6_addr8
#else /* NB: BSD/OSX may require others */
#define s6_addr8 __u6_addr.__u6_addr8
#define s6_addr32 __u6_addr.__u6_addr32
#endif	/* __linux__ */

#define MASKEQUAL(x,y,z) (\
	(((x)->s6_addr32[0] & (y)->s6_addr32[0]) == (z)->s6_addr32[0]) && \
	(((x)->s6_addr32[1] & (y)->s6_addr32[1]) == (z)->s6_addr32[1]) && \
	(((x)->s6_addr32[2] & (y)->s6_addr32[2]) == (z)->s6_addr32[2]) && \
	(((x)->s6_addr32[3] & (y)->s6_addr32[3]) == (z)->s6_addr32[3]))

#define	SETADR6(a)		\
	packadrinfo(AF_INET6, (u_char *)&(a), cd.address)

#endif	/* SUBCALC_DOT_H_ */
