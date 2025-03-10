FROM alpine:3.18

WORKDIR /app
COPY kube-registry-bouncer /app/

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -u 1000 -G appuser -s /bin/sh -D appuser && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 1323

ENTRYPOINT ["/app/kube-registry-bouncer"]
