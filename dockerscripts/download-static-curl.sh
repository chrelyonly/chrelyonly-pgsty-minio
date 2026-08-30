#!/bin/bash

# Pin to v8.11.0 – the last release that includes curl-aarch64.
# v8.17.0 (current latest) dropped the aarch64 build.
STATIC_CURL_VERSION="v8.11.0"

case ${TARGETARCH:?TARGETARCH is required} in
amd64)
	asset=curl-amd64
	expected=d18aa1f4e03b50b649491ca2c401cd8c5e89e72be91ff758952ad2ab5a83135d
	;;
arm64)
	asset=curl-aarch64
	expected=1b050abd1669f9a2ac29b34eb022cdeafb271dce5a4fb57d8ef8fadff6d7be1f
	;;
*)
	echo "Unsupported static cURL architecture: ${TARGETARCH}" >&2
	exit 1
	;;
esac

curl --fail --location --silent --show-error --retry 3 \
	"https://github.com/moparisthebest/static-curl/releases/download/${STATIC_CURL_VERSION}/${asset}" \
	--output /go/bin/curl
printf '%s  %s\n' "${expected}" /go/bin/curl | sha256sum -c
chmod +x /go/bin/curl
