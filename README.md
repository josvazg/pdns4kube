# pdns4kube

A Kubernetes operator that manages DNS zones in [PowerDNS](https://www.powerdns.com/) Authoritative Server.
It watches `DNSZone` resources (`pdns.example.io/v1`) and synchronizes zone create/update/delete
with the PowerDNS Authoritative API.

## Prerequisites

- [Go](https://go.dev/)
- [Docker](https://www.docker.com/)
- [Kind](https://kind.sigs.k8s.io/)
- [kubectl](https://kubernetes.io/docs/reference/kubectl/)
- [Mage](https://magefile.org/)

Alternatively, if you use [Nix](https://nixos.org/), you can obtain the dev tools
with `nix develop` instead of installing them individually.

## Quick start

Start everything (Kind cluster, CRDs, a dev PowerDNS, API port-forward, and the operator).

With Nix, enter the dev shell:

```sh
nix develop
```

Then, from the resulting dev shell, run:

```sh
mage run
```

Without Nix (tools installed individually):

```sh
mage run
```

This creates a Kind cluster named `pdns4kube`, installs the CRD, deploys a dev PowerDNS,
forwards its API to `http://localhost:18081`, and runs the operator locally. The operator
is configured via the `PDNS_API_URL` and `PDNS_API_KEY` environment variables (set
automatically by `mage run`).

While `mage run` is running, create a zone:

```sh
kubectl apply -f - <<'EOF'
apiVersion: pdns.example.io/v1
kind: DNSZone
metadata:
  name: example-org
spec:
  v0_0_15:
    entry:
      kind: Native
      name: example.org.
      nameservers:
        - ns1.example.org.
        - ns2.example.org.
EOF
```

Verify the zone exists in PowerDNS:

```sh
kubectl get dnszones
curl -s -H "X-API-Key: pdns4kube-dev-key" http://localhost:18081/api/v1/servers/localhost/zones
```

## Cleanup

- `mage pdnsDown` — removes the dev PowerDNS resources but keeps the Kind cluster.
- `mage kindDown` — deletes the whole Kind cluster.

## Notes

- The dev API key `pdns4kube-dev-key` is for local development only and is not secure.
- When running the operator manually, configure it with:
  - `PDNS_API_URL` — PowerDNS API endpoint (e.g. `http://127.0.0.1:18081`)
  - `PDNS_API_KEY` — PowerDNS API key
