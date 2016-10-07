# dns-filter

The inspiration for creating dns-filter comes from a decision that Netflix recently made to [block users of Hurricane Electric's](https://torrentfreak.com/netflix-blocks-ipv6-over-geo-unblocking-fears-160608/) free [IPv6 over IPv4 tunnels](https://tunnelbroker.net "tunnelbroker.net"). 

By blocking IPv6 DNS resolution from Netflix, Netflix bound traffic will no longer go across Hurricane Electric's network as it will be restricted to IPv4 only. This is a better solution than [null routing all AWS IPv6 traffic](https://forums.he.net/index.php?topic=3564.msg20354#msg20354)

dns-filter works by allowing users to define which dns results to block. To do this, dns-filter can be used as a dns proxy sitting between your machine or network and an upstream dns server. 

An example of using unbound to delegate dns resolution to dns-filter
``` sh
forward-zone:
        name: "netflix.com"
        forward-addr: 127.0.0.1@1234
        forward-first: yes
```
For example, since Netflix works over both IPv4 **and** IPv6, a configuration can be setup to block any IPv6 dns responses from Netflix like so:

``` json
"filters" : [
        {
            "host": "netflix.com",
            "type": "AAAA",
            "matching": "contains"
        }
    ]
```

Launching dns-filter can be done in a script or the command line:
``` sh
./filter-dns -c conf/config.json 
```
## Configuration
Configuring dns-filter is as easy as creating a json file. A sample configuration file has been provided in the **conf** directory:

``` json
{
    "host" : "localhost",
    "port" : 1234,
    "logfile": "dns-filter.log",
    "forwarders" : [
        {
            "address": "74.82.42.42",
            "port": 53,
            "protocol": "udp"
        },
        {
            "address": "37.235.1.174",
            "port": 53,
            "protocol": "udp"
        },
        {
            "address": "107.150.40.234",
            "port": 53,
            "protocol": "udp"
        },
        {
            "address": "8.8.8.8",
            "port": 53,
            "protocol": "udp"
        }
    ],
    "filters" : [
        {
            "host": "netflix.com",
            "type": "AAAA",
            "matching": "contains"
        }
    ]
}
```
With the above example, dns-filter will:

* Listen on **localhost** on port **1234**
* Create a log file called **dns-filter.log** in the same working directory that launches dns-filter
* Define a list of dns forwarders with their respective ip addresses, port numbers and protocol (**udp/tcp**). 
* Block any ipv6 (**AAAA**) dns responses for the entire netflix.com domain

## What dns-filter doesn't do
(currently)

* Cache DNS results
* DNSSEC validation
* Access control lists

## Building
To build dns-filter, first download all dependencies with
``` sh
go get
```
and then 
``` sh
make build
```
to build the actual binary. To build for other platforms, run the `build-x` target:

```console
make build-x
```

The resulting binaries will be in the `bin/` subdirectory.

## Big Thanks
I would like to thank [Mateusz Kaczanowski](http://mkaczanowski.com) for publishing an [example](http://mkaczanowski.com/golang-build-dynamic-dns-service-go/ "#Build your own dynamic DNS service with GO!") that I used to get familiar with the [Golang DNS library](https://github.com/miekg/dns)

And also a big shout out to [Miek Gieben](https://github.com/miekg) for creating the [Golang DNS library](https://github.com/miekg/dns)
