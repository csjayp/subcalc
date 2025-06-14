# subcalc API module for Fastly

Build a Fastly compute service which wraps the subcalc golang package bindings.

## Building

This service can be built using `make` or the `fastly compute build` command. If you want to run an instance of this service locally, you can do so through the Fastly Viceroy utility which allows you to run WASM packages on your host. This can be done using `make run-local` or `fastly compute serve`.

## Testing

This service runs on Fastly compute @edge and provides the same services that the `subcalc` command line tools provides. Fllowing are some basic usages. The first example illustrates how retrieve basic subnet calculation information for an IP4 address:

```
csjp@vmxon ~ % curl -s https://api.sqrt.ca/inet/10.0.0.1/24 | jq .
{
  "subcalc_query": {
    "address_family": 0,
    "address": "10.0.0.1",
    "cidr_bits": 24,
    "print_list": false
  },
  "subcalc_answer": {
    "address_range": {
      "first_address": "10.0.0.0",
      "last_address": "10.0.0.255"
    },
    "address_range_base10": {
      "first_address": "167772160",
      "last_address": "167772415"
    },
    "address_range_base16": {
      "first_address": "0xa000000",
      "last_address": "0xa0000ff"
    },
    "host_count": "256",
    "prefix_length": 24,
    "network_mask": "255.255.255.0",
    "mask": "0.0.0.255"
  }
}
```

The API also supports printing the networks based on the provided information (similar to the command line `print` option):

```
csjp@vmxon ~ % curl -s https://api.sqrt.ca/inet/10.0.0.1/30/print | jq .
{
  "subcalc_query": {
    "address_family": 0,
    "address": "10.0.0.1",
    "cidr_bits": 30,
    "print_list": true
  },
  "subcalc_answer": {
    "address_range": {
      "first_address": "10.0.0.0",
      "last_address": "10.0.0.3"
    },
    "address_range_base10": {
      "first_address": "167772160",
      "last_address": "167772163"
    },
    "address_range_base16": {
      "first_address": "0xa000000",
      "last_address": "0xa000003"
    },
    "host_count": "4",
    "prefix_length": 30,
    "network_mask": "255.255.255.252",
    "mask": "0.0.0.3"
  },
  "net_list": [
    "10.0.0.0",
    "10.0.0.1",
    "10.0.0.2",
    "10.0.0.3"
  ]
}
csjp@vmxon ~ % 
```

The following illustraces the same some functionality but with an IP6 address:

```
csjp@vmxon ~ % curl -s https://api.sqrt.ca/inet6/2002:dead:beef::1/120 | jq .
{
  "subcalc_query": {
    "address_family": 1,
    "address": "2002:dead:beef::1",
    "cidr_bits": 120,
    "print_list": false
  },
  "subcalc_answer": {
    "address_range": {
      "first_address": "2002:dead:beef::",
      "last_address": "2002:dead:beef::ff"
    },
    "address_range_base10": {
      "first_address": "",
      "last_address": ""
    },
    "address_range_base16": {
      "first_address": "",
      "last_address": ""
    },
    "host_count": "256",
    "prefix_length": 120,
    "network_mask": "ffffffffffffffffffffffffffffff00",
    "mask": "::ff"
  }
}
csjp@vmxon ~ % 
```

More the same with the `print` features. Note care should be taken when using the `/print` endpoint with IP6 networks as the amount of data returned by the API could be huge:

```
csjp@vmxon ~ % curl -s https://api.sqrt.ca/inet6/2002:dead:beef::1/127/print | jq .
{
  "subcalc_query": {
    "address_family": 1,
    "address": "2002:dead:beef::1",
    "cidr_bits": 127,
    "print_list": true
  },
  "subcalc_answer": {
    "address_range": {
      "first_address": "2002:dead:beef::",
      "last_address": "2002:dead:beef::1"
    },
    "address_range_base10": {
      "first_address": "",
      "last_address": ""
    },
    "address_range_base16": {
      "first_address": "",
      "last_address": ""
    },
    "host_count": "2",
    "prefix_length": 127,
    "network_mask": "fffffffffffffffffffffffffffffffe",
    "mask": "::1"
  },
  "net_list": [
    "2002:dead:beef::",
    "2002:dead:beef::1"
  ]
}
csjp@vmxon ~ % 
```
