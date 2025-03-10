#!/bin/bash
set -e

NAMESPACE="kube-registry-bouncer-poc"
SECRET_NAME="kube-registry-bouncer-tls-secret"

# Create certificates
echo "📜 Creating TLS certificates..."
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout tls.key -out tls.crt \
  -subj "/CN=kube-registry-bouncer-server.${NAMESPACE}.svc" \
  -addext "subjectAltName = DNS:kube-registry-bouncer-server.${NAMESPACE}.svc"

# Apply namespace and deployment
echo "🚀 Deploying namespace, service and deployment..."
kubectl apply -f kubernetes/deployment.yaml
kubectl apply -f kubernetes/service.yaml

# Create TLS secret
echo "🔒 Creating TLS secret..."
kubectl create secret tls ${SECRET_NAME} \
  -n ${NAMESPACE} \
  --cert=tls.crt --key=tls.key \
  --dry-run=client -o yaml | kubectl apply -f -

# Wait for deployment to be ready
echo "⏳ Waiting for deployment to be ready..."
kubectl rollout status deployment/kube-registry-bouncer-server -n ${NAMESPACE} --timeout=60s

# Create the webhook with CA Bundle
echo "🔗 Configuring webhook..."
CA_BUNDLE=$(cat tls.crt | base64 | tr -d '\n')
sed "s|\${CA_BUNDLE}|${CA_BUNDLE}|g" kubernetes/webhook.yaml | kubectl apply -f -

echo "✅ Installation complete!"