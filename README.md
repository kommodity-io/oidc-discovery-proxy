# OIDC Discovery Proxy

A lightweight proxy server that enables external access to Kubernetes cluster's OpenID Connect (OIDC) discovery endpoints. This allows workload identity federation and external service authentication with Kubernetes service accounts.

## Overview

The OIDC Discovery Proxy exposes two critical OIDC endpoints from your Kubernetes cluster:
- `/.well-known/openid-configuration` - OpenID Connect discovery document
- `/openid/v1/jwks` - JSON Web Key Set (JWKS) for token verification

This enables external systems (such as cloud providers, CI/CD platforms, or other services) to validate Kubernetes service account tokens issued by your cluster.

## Features

- **Lightweight**: Minimal resource footprint with efficient Go implementation
- **Secure**: Direct proxy to Kubernetes API server endpoints
- **Flexible Deployment**: Support for both Gateway API and traditional Ingress
- **Cloud Native**: Packaged as a container and distributed via Helm chart
- **Multi-Architecture**: Supports `linux/amd64` and `linux/arm64`

## Use Cases

### Workload Identity Federation

Configure your cloud provider (AWS, GCP, Azure) to trust your cluster's OIDC issuer:

1. Deploy OIDC Discovery Proxy with a public endpoint
2. Configure your cloud IAM to trust the issuer URL
3. Service accounts can now authenticate to cloud services

### Cross-Cluster Authentication

Enable service-to-service authentication across multiple clusters.

## Architecture

```
┌─────────────┐         ┌──────────────────┐         ┌─────────────────┐
│   Client    │────────>│  OIDC Discovery  │────────>│  Kubernetes API │
│             │         │      Proxy       │         │     Server      │
└─────────────┘         └──────────────────┘         └─────────────────┘
                              │
                              │ Proxies:
                              │ - /.well-known/openid-configuration
                              │ - /openid/v1/jwks
```

The proxy runs as a lightweight Go application that forwards requests to the Kubernetes API server's OIDC endpoints.

## Security Considerations

- The proxy uses in-cluster authentication to access the Kubernetes API
- No sensitive data is stored or cached
- Requests are directly proxied to the API server
- TLS termination should be handled by your Ingress/Gateway
- Consider rate limiting at the Ingress/Gateway level

## Installation

### Using Helm

Add the Helm repository:

```bash
helm repo add kommodity https://ghcr.io/kommodity-io/charts
helm repo update
```

Install the chart:

```bash
helm install oidc-discovery-proxy kommodity/oidc-discovery-proxy \
  --namespace oidc-discovery-proxy \
  --create-namespace \
  --set host.domain=example.com
```

### Using Gateway API

```bash
helm install oidc-discovery-proxy kommodity/oidc-discovery-proxy \
  --namespace oidc-discovery-proxy \
  --create-namespace \
  --set gateway.enabled=true \
  --set gateway.name=envoy-gateway \
  --set gateway.namespace=default \
  --set host.domain=example.com \
  --set host.prefixes={cluster1,cluster2}
```

This will expose the endpoints at:
- `https://cluster1.example.com/.well-known/openid-configuration`
- `https://cluster1.example.com/openid/v1/jwks`
- `https://cluster2.example.com/.well-known/openid-configuration`
- `https://cluster2.example.com/openid/v1/jwks`

### Using Ingress

```bash
helm install oidc-discovery-proxy kommodity/oidc-discovery-proxy \
  --namespace oidc-discovery-proxy \
  --create-namespace \
  --set gateway.enabled=false \
  --set ingress.className=nginx \
  --set host.domain=example.com \
  --set host.exact={oidc.example.com}
```

## Configuration

### Chart Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `gateway.enabled` | Use Gateway API instead of Ingress | `true` |
| `gateway.name` | Name of the Gateway resource | `envoy-gateway` |
| `gateway.namespace` | Namespace of the Gateway resource | `default` |
| `ingress.className` | Ingress class name | `""` |
| `host.domain` | Base domain for the proxy | `REPLACE_ME.com` |
| `host.prefixes` | List of subdomain prefixes | `None` |
| `host.exact` | List of exact hostnames | `None` |
| `image.repository` | Container image repository | `ghcr.io/kommodity-io/oidc-discovery-proxy` |
| `image.tag` | Container image tag | `v0.1.0` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `replicas` | Number of replicas | `2` |
| `resources.requests.cpu` | CPU request | `50m` |
| `resources.requests.memory` | Memory request | `32Mi` |
| `resources.limits.cpu` | CPU limit | `200m` |
| `resources.limits.memory` | Memory limit | `64Mi` |

### Example: Custom Configuration

```yaml
gateway:
  enabled: false

ingress:
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod

host:
  domain: k8s.example.com
  exact:
    - prod-cluster.example.com
    - staging-cluster.example.com

resources:
  requests:
    cpu: 100m
    memory: 64Mi
  limits:
    cpu: 500m
    memory: 128Mi

replicas: 3
```

## Development

### Building from Source

Requirements:
- Go 1.25+
- Make
- Docker (for container builds)
- UPX (optional, for binary compression)

Build the binary:

```bash
make build
```

Run locally:

```bash
make run
```

Build container image:

```bash
make build-image
```

### Linting

```bash
make lint
```

Fix linting issues:

```bash
make lint-fix
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Links

- **Container Images**: [ghcr.io/kommodity-io/oidc-discovery-proxy](https://github.com/kommodity-io/oidc-discovery-proxy/pkgs/container/oidc-discovery-proxy)
- **Helm Charts**: [ghcr.io/kommodity-io/charts](https://github.com/orgs/kommodity-io/packages)
- **Issues**: [GitHub Issues](https://github.com/kommodity-io/oidc-discovery-proxy/issues)
