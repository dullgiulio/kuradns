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

