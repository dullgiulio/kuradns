# KuraDNS - DNS server with REST-ish interface

## Usage

Using bat:

```
$ bat localhost:8080/source/add \
	source.name=domains \
	source.type=mysql \
	config.user=root \
	config.password='rootpass' \
	config.database=domainsdb \
	config.query="SELECT domainName, 'canonicalname.org' FROM domains WHERE ..."
$ bat localhost:8080/dns/dump
```

Adding a single entry:
```
$ bat localhost:8080/source/add \
	source.name=entry0 \
	source.type=static \
	config.key=mydamin.local \
	config.val=127.0.0.1
```

## Setup

To listen on standard DNS port 53, use:
```
# setcap 'cap_net_bind_service=+ep' kuradns
```

Run:
```
$ kuradns -info -zone myzone.lan -dns 0.0.0.0:53
```

Might be necessary to listen to another port and redirect external traffic to this port:
```
# iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 53 -j REDIRECT --to-port 8053
```

## Wildcards

Wildcards are supported as targets. Wildcards are flattened out, ignoring all DNS standards.

For example "something.*.local" as target will be matched by "something.hello.local". Other allowed patterns are:
```
a*.
*a.
a*a.
```

Only one star is allowed but in any position or as a whole level of the DNS entry.
Any level could be a wildcard but zone must be respected.
