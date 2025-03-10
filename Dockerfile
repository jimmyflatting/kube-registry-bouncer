FROM alpine:3.17

COPY /kube-registry-bouncer /usr/local/bin/kube-registry-bouncer
RUN chmod +x /usr/local/bin/kube-registry-bouncer

ENTRYPOINT ["kube-registry-bouncer"]
