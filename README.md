# subcalc #

Subnet calculation and discovery utility for BSD & Linux.

## Overview ##

The subcalc utility is used for subnet calculation and IPv6 DNS PTR
record generation.  subcalc takes command line arguments in a similar
format to ifconfig(8) so the synopsis should be familiar to the user.
Given an address family, address and a netmask/prefix length, subcalc
will calculate the number of hosts and address ranges of the specific
network. Specifying the ``print'' option to to the end of the command
line will result in every single network address for the specified net-
work being printed to stdout.

subcalc was designed for network engineers which setup fine grained firewalls, packet filters, access control lists and network subdivisions for both IP and IPv6 servers and networks.

## Usage ##

	usage:  subcalc [inet | inet6] address [netmask | mask ] mask <print>
		    subcalc [inet | inet6] address [prefixlen] len <print>
		    subcalc [inet | inet6] hosts value
		    subcalc [int6 | arpa6] address hostname
		    subcalc stf [inet | inet6 ] address

## Examples ##

To calculate the network range, number of hosts, prefixlen or CIDR and netmask for the 10.0.0.1/24 (255.255.255.0) network.

           % subcalc inet 10.0.0.1/24

Anyone of the following will achieve the exact same thing:

           % subcalc inet 10.0.0.1 netmask 255.255.255.0
           % subcalc inet 10.0.0.1 netmask 0xffffff00
           % subcalc inet 10.0.0.1 prefixlen 24

To generate a list of nodes for the specified network one could use anyone of the following methods:

           % subcalc inet 10.0.0.1/24 print
           % subcalc inet 10.0.0.1 netmask 255.255.255.0 print
           % subcalc inet 10.0.0.1 netmask 0xffffff00 print
           % subcalc inet 10.0.0.1 prefixlen 24 print

Arbitrarily, the same thing can be done for IPv6. To calculate the network range, number of hosts, prefixlen etc for the 3ffe:beef:13e1:4c92::cd90/48 network, one could use any of the following:

           % subcalc inet6 3ffe:beef:13e1:4c92::cd90/48
           % subcalc inet6 3ffe:beef:13e1:4c92::cd90 netmask ffff:ffff:ffff::
           % subcalc inet6 3ffe:beef:13e1:4c92::cd90 prefixlen 48

## License ##

	Copyright (c) Christian S.J. Peron (csjp@sqrt.ca) 
	All rights reserved.

	Redistribution and use in source and binary forms, with or without
	modification, are permitted provided that the following conditions
	are met:
	1. Redistributions of source code must retain the above copyright
	   notice, this list of conditions and the following disclaimer.
	2. Redistributions in binary form must reproduce the above copyright
	   notice, this list of conditions and the following disclaimer in the
	   documentation and/or other materials provided with the distribution.
	3. The names of the authors may not be used to endorse or promote
	   products derived from this software without specific prior written
	   permission.

	THIS SOFTWARE IS PROVIDED BY THE AUTHOR AND CONTRIBUTORS ``AS IS'' AND
	ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
	IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
	ARE DISCLAIMED.  IN NO EVENT SHALL THE AUTHOR OR CONTRIBUTORS BE LIABLE
	FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
	DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
	OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
	HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
	LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
	OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
	SUCH DAMAGE.
