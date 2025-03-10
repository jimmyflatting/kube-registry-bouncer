# kube-registry-bouncer
A Kubernetes admission controller webhook that validates image registries used in pods and rejects deployments using unauthorized registries.

## Overview
kube-registry-bouncer is a validating webhook for Kubernetes that ensures all container images come from trusted registries. It helps maintain security and compliance by preventing the use of unapproved container registries in your cluster.

## Features
- Validates all container images against a configurable whitelist of approved registries
- Rejects pods that use images from unauthorized sources
- Simple configuration via environment variables or command-line flags
- Detailed logging for auditing and troubleshooting
- Kubernetes-native integration via validating webhook

### Installation
#### Prerequisites
- Kubernetes cluster with admission controllers enabled
- kubectl configured to communicate with your cluster
- OpenSSL for generating certificates

#### Quick Install
1. Clone the repository:
```bash
git clone https://github.com/jimmyflatting/kube-registry-bouncer.git
cd kube-registry-bouncer
```

2. Run the installation script:
```bash
chmod +x install.sh
./install.sh
```
The install script will:

* Generate TLS certificates
* Create necessary Kubernetes resources
* Set up the ValidatingWebhookConfiguration

#### Manual Installation
1. Generate TLS certificates
```bash
# Create certificates for the webhook server
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout tls.key -out tls.crt \
  -subj "/CN=kube-registry-bouncer-server.kube-registry-bouncer-poc.svc" \
  -addext "subjectAltName = DNS:kube-registry-bouncer-server.kube-registry-bouncer-poc.svc"

# Create Kubernetes namespace
kubectl apply -f kubernetes/deployment.yaml

# Create TLS secret
kubectl create secret tls kube-registry-bouncer-tls-secret \
  -n kube-registry-bouncer-poc \
  --cert=tls.crt --key=tls.key
```
2. Deploy the service
```bash
kubectl apply -f kubernetes/service.yaml
```
3. Deploy the webhook configuration
```bash
CA_BUNDLE=$(cat tls.crt | base64 | tr -d '\n')
sed "s|\${CA_BUNDLE}|${CA_BUNDLE}|g" kubernetes/webhook.yaml | kubectl apply -f -
```

### Configuration
#### Registry Whitelist
Configure the allowed registries by setting the `KUBE_BOUNCER_REGISTRY_WHITELIST` environment variable in the deployment:
```bash
env:
- name: KUBE_BOUNCER_REGISTRY_WHITELIST
  value: "ghcr.io,docker.io,quay.io" # Comma-separated list
```
#### TLS Certificate and Key
The webhook requires TLS certificates to secure communication:
```bash
env:
- name: KUBE_BOUNCER_CERTIFICATE
  value: "/etc/kube-registry-bouncer/certs/tls.crt"
- name: KUBE_BOUNCER_KEY
  value: "/etc/kube-registry-bouncer/certs/tls.key"
```
#### Debug Mode
Enable debug logging for more verbose output:
```bash
env:
- name: KUBE_BOUNCER_DEBUG
  value: "true"
```

### Usage
After installation, the webhook will automatically validate all pod creation and update requests in your cluster, except in the namespaces excluded in the webhook configuration.

#### Testing the Webhook
Create a test deployment using an allowed registry:
```bash
kubectl apply -f kubernetes/tests/deployment-success.yaml
```
Create a test deployment using a blocked registry:
```bash
kubectl apply -f kubernetes/tests/deployment-fail.yaml
```
Check the logs to see the validation results:
```bash
kubectl logs -n kube-registry-bouncer-poc -l app=kube-registry-bouncer-server
```
### Development
Building from Source
```bash
# Build locally
go build -o kube-registry-bouncer main.go

# Run locally (for testing)
./kube-registry-bouncer --registry-whitelist=ghcr.io
```
Docker Build
```bash
docker build -t ghcr.io/jimmyflatting/kube-registry-bouncer:dev .
```
### Release Process
The project uses GoReleaser with GitHub Actions for automated releases:
1. Tag a new version:
```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```
2. GitHub Actions will automatically:

* Build the binary
* Create a Docker image
* Push to GitHub Container Registry
* Create a GitHub release

### License
MIT License

### Contributing
Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (git checkout -b feature/amazing-feature)
3. Commit your changes (git commit -m 'Add some amazing feature')
4. Push to the branch (git push origin feature/amazing-feature)
5. Open a Pull Request