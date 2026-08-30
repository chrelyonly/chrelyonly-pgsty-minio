# Client source address trust

Silo decides where a request came from, and that decision is load-bearing. The
address it settles on becomes:

- `aws:SourceIp`, so it decides `IpAddress` and `NotIpAddress` policy conditions
- the audit log's `remotehost` field, so it decides who every logged action is
  attributed to
- the `Host` field of S3 event notifications, and the client shown by
  `mc admin trace`

None of that is derived from the TCP connection by default. It is read out of the
`X-Forwarded-For`, `X-Real-IP` and RFC 7239 `Forwarded` request headers, which
any client can set to any value. This document states which peers Silo believes,
how to change that, and what each choice costs.

## The three modes

One setting selects all three. It is read from the environment, not from
`mc admin config`.

| Mode | `MINIO_API_TRUSTED_PROXIES` | Source address |
| :-- | :-- | :-- |
| Untrusted (default) | unset | left-most `X-Forwarded-For` entry, else `X-Real-IP`, else `Forwarded`, else the TCP peer — from any client |
| Trust nobody | `none` | always the TCP peer |
| Allow-listed | a list of addresses and CIDR blocks | forwarded headers, but only from listed peers |

`off` is accepted as a synonym for `none`, and the value is matched
case-insensitively.

The setting is read once at startup, after `MINIO_CONFIG_ENV_FILE` is loaded, so
an environment file is a valid place to put it. A malformed value stops the
server rather than silently changing what every policy condition resolves to.

> **`_MINIO_API_XFF_HEADER` is not this setting.** It suppresses parsing of
> `X-Forwarded-For` and nothing else, leaving `X-Real-IP` and `Forwarded`
> honoured, so it cannot stop a client naming its own address — a client refused
> one header simply sends another. It keeps its original upstream meaning, and
> applies *within* whichever mode above is in force. If you set it hoping to stop
> source-address forgery, that is what `MINIO_API_TRUSTED_PROXIES` is for.
>
> It also keeps upstream's read timing, which is *earlier* than the one described
> above: it is taken at package initialisation, before environment files are
> read, so writing it into `MINIO_CONFIG_ENV_FILE` has no effect. That quirk is
> deliberately left in place rather than repaired, because repairing it would
> make an already-deployed setting start taking effect. Set it in the process
> environment if you want it honoured.

### Untrusted (default)

Every deployment behaves exactly as it did before this setting existed. Any
client that can open a connection to the S3 API port can name its own address,
so under this mode:

**An `IpAddress` policy condition is not enforceable, and audit client addresses
are not evidence.** Both are attacker-chosen for anyone with direct network
access to the API port.

This mode is sound only when every route to the API port passes through a proxy
that overwrites all three headers. Two things commonly break that assumption:

- **A second way in.** On Kubernetes an Ingress and a ClusterIP Service usually
  coexist. The Ingress sanitises headers; the Service does not, and any pod in
  the cluster can reach it. The boundary is assumed to be the Ingress but is
  actually the pod network.
- **Appending proxies.** The stock nginx recipe
  `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;` *appends* to
  what the client sent. A client sending `X-Forwarded-For: 1.2.3.4` produces
  `1.2.3.4, <real client>` at Silo, and this mode reads the left-most entry —
  the client's. Overwriting with `$remote_addr` avoids this; so does the
  allow-listed mode below, which reads the chain from the other end.

### Trust nobody

```
MINIO_API_TRUSTED_PROXIES=none
```

No forwarded header is believed. The source address is the TCP peer, always.

Use this when Silo is reached directly and you would rather have a correct
address that is sometimes a proxy than a plausible one that is sometimes a lie.
Behind a proxy, every request will be attributed to the proxy.

**On a multi-node deployment this also applies to Silo's own nodes.** Some
requests are forwarded between nodes (see [Multi-node](#multi-node-deployments)),
and the receiving node's peer is the forwarding node, so those requests are
attributed to it rather than to the client. Nothing corrects this under this
mode, because it believes nothing. Multi-node clusters that want direct requests
protected *and* internally-forwarded ones attributed correctly should use the
allow-listed mode with the node addresses included.

The scheme headers (`X-Forwarded-Proto`, `X-Forwarded-Scheme`) are deliberately
unaffected by any of this. They feed the `Location` URL in S3 responses, not a
policy decision, and suppressing them would hand `http://` URLs to every
deployment terminating TLS at a proxy.

### Allow-listed

```
MINIO_API_TRUSTED_PROXIES=10.0.0.1,10.0.0.2
```

A comma-, semicolon- or whitespace-separated list of addresses and CIDR blocks.
Forwarded headers are believed only when the request's TCP peer matches. Every
other peer is attributed to itself, whatever its headers claim.

Write entries in their plain form. An address written in IPv4-mapped notation
(`::ffff:192.168.1.10`) is accepted but matches nothing, because peers are
reduced to plain form before matching — so the entry grants no trust and the
proxy is treated as any other client. This is long-standing behaviour shared
with `MINIO_IDENTITY_LDAP_STS_TRUSTED_PROXIES`, and it fails closed.

This is the only mode under which `aws:SourceIp` is enforceable against a client
that can reach the API port directly.

Catch-all ranges (`0.0.0.0/0`, `::/0`) are rejected. They would trust every peer
and quietly reinstate the untrusted mode under a name that suggests otherwise.
Note that only `/0` itself is rejected — a pair like `0.0.0.0/1,128.0.0.0/1`
covers the same ground and is accepted, so the check is a guardrail, not a proof.

> **List proxies, not the subnet they sit in.** This is the one way to configure
> the setting so that it makes things worse rather than better, so it is worth
> stating plainly.
>
> Entries on the list are skipped during the chain walk described below. If the
> list covers addresses that *clients* also occupy — `10.0.0.0/8` when the load
> balancer is `10.0.0.1` and application servers are elsewhere in `10/8` — then
> those clients' addresses are skipped too, and the walk continues past the real
> client into whatever it placed to the left. A client at `10.5.5.5` sending
> `X-Forwarded-For: 8.8.8.8` would then be recorded as `8.8.8.8`.
>
> A broad list does not merely trust more peers; it lets those peers forge.
> Prefer exact host entries or the narrowest prefix that contains only proxies.

**Chain handling.** `X-Forwarded-For` and `Forwarded` are read right-to-left,
stepping over entries that name a listed proxy, and the first remaining address
wins. Each proxy appends the peer it actually saw, so an entry the client
injected sits to the left of the one its proxy wrote, and the walk stops before
reaching it. Appending proxies are therefore safe here. List every proxy hop's
address so intermediate hops are skipped. Repeated header lines are handled as
one chain, so proxies that add a second `X-Forwarded-For` line rather than
extending the first — HAProxy's `option forwardfor` — work correctly.

**`X-Real-IP` is a single value with no chain, so it cannot be checked against
the list.** It is trusted verbatim, and only when the chain headers yield
nothing. The deployment contract is that a listed proxy overwrites whichever
headers it sets — for nginx, `proxy_set_header X-Real-IP $remote_addr;`. A proxy
that relays the client's copy instead is choosing to let the client answer the
question, and nothing Silo does can undo that.

> **Strip at the edge every source-address header your proxy does not itself
> write.** This is the one rule that covers every case, and it is worth following
> even if the rest of this section is skipped.
>
> Listing a peer means believing all three headers from it, and they are
> consulted in order: `X-Forwarded-For`, then `X-Real-IP`, then `Forwarded`. A
> header your proxy does not write is entirely under the client's control, and if
> it is consulted before the one your proxy *does* write, the client wins.
>
> The common way to get bitten: a proxy that authors only `X-Real-IP` (some nginx
> configurations) or only `Forwarded` (RFC 7239-native proxies) relays the
> client's `X-Forwarded-For` untouched — and `X-Forwarded-For` is read first, so
> a client sending `X-Forwarded-For: 8.8.8.8` is recorded as `8.8.8.8` despite
> the proxy having correctly written the real address elsewhere.

(`MINIO_IDENTITY_LDAP_STS_TRUSTED_PROXIES` orders the first two the other way
round. Neither order is safe for every proxy — the mirror hazard is a proxy that
authors only `X-Forwarded-For` and relays a client's `X-Real-IP`, which is what
AWS ALB does. This setting prefers the header it can chain-validate, because it
decides access control rather than rate-limit bucketing. Stripping what your
proxy does not write makes the ordering irrelevant, which is why it is the rule
worth remembering.)

**Loopback is always trusted as a peer**, even when not listed. The FTP and SFTP
front-ends reach the S3 layer over `127.0.0.1` and declare their session's client
with `X-Forwarded-For`; excluding loopback would attribute every FTP and SFTP
request to the server itself. A `127.0.0.1` entry appearing *inside* a chain is
not skipped — only peers are exempt, not chain entries.

## Multi-node deployments

**A cluster must list its own nodes.** Silo forwards some requests between
nodes — bucket-DNS and site-replication routing, listing continuation,
heal-by-token, batch jobs and pool decommissioning. The receiving node's peer is
the forwarding node, so unless the cluster's own addresses are on the list, those
requests resolve to the forwarding node rather than to the client. Include the
node addresses alongside your proxy's.

This is not a rare path. A `ListObjectsV2` continuation token carries the node
index, so any client can cause its own request to be forwarded. If the nodes are
not listed, those requests are evaluated with `aws:SourceIp` set to an internal
node address and audited against one — which an `IpAddress` condition that allows
internal ranges would treat as a pass.

## Choosing

Behind a proxy you control, with no other route to the API port, the default is
fine and you need not set anything — but confirm the proxy overwrites rather than
appends, or you are in the second case below. Otherwise:

- Direct exposure, no proxy: `MINIO_API_TRUSTED_PROXIES=none`.
- Behind a proxy, but the port is also reachable directly (the usual Kubernetes
  Ingress-plus-Service case, and the usual Pigsty case): set
  `MINIO_API_TRUSTED_PROXIES` to the proxy addresses plus the Silo node
  addresses. This is the configuration that makes an `IpAddress` condition mean
  something.
- Multi-node clusters: use the allow-list with node addresses included, not
  `none`.
- Behind a proxy you are not sure about: allow-list it, and strip at the edge
  every source-address header it does not itself write.

Whichever applies, two rules govern every allow-list deployment: name proxies,
not the subnet they occupy; and strip what your proxy does not write.

## Relationship to LDAP STS trusted proxies

`MINIO_IDENTITY_LDAP_STS_TRUSTED_PROXIES` is a separate allow-list governing
which peers may supply the client address used for **LDAP STS login rate-limit
bucketing** only. It does not affect `aws:SourceIp` or audit addresses, and this
setting does not affect STS rate limiting. The two use the same list syntax and
the same chain-walking rules, and in most deployments should be set to the same
value. See [`docs/sts/ldap.md`](../sts/ldap.md).
