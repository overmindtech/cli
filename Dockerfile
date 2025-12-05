FROM ghcr.io/opentofu/opentofu:minimal AS tofu

FROM alpine:3.23.0

# Copy the tofu binary from the minimal image
COPY --from=tofu /usr/local/bin/tofu /usr/local/bin/tofu

# Add the Overmind public key directly
ADD https://dl.cloudsmith.io/public/overmind/tools/rsa.7B6E65C2058FDB78.key \
    /etc/apk/keys/tools@overmind-7B6E65C2058FDB78.rsa.pub

# Add repository config
ADD https://dl.cloudsmith.io/public/overmind/tools/config.alpine.txt?distro=alpine&codename=v3.8 \
    /tmp/config.alpine.txt
RUN cat /tmp/config.alpine.txt >> /etc/apk/repositories \
    && rm /tmp/config.alpine.txt

RUN apk update
RUN apk add --no-cache overmind-cli
