FROM alpine:3.17
ARG TARGETARCH
COPY dist/kube-registry-bouncer_linux_${TARGETARCH}*/kube-registry-bouncer /usr/local/bin/
RUN chmod +x /usr/local/bin/kube-registry-bouncer
CMD ["/usr/local/bin/kube-registry-bouncer"]
