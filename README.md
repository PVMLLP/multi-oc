# multi-oc (moc)

Central CLI for OpenShift multi-cluster environments (no ACM proxy required). Uses the hub cluster for discovery and talks directly to managed clusters using your SSO/token. Designed for RHEL jump hosts with direct network access to the hub and all managed clusters.

## Features
- Interactive hub login (prompts for hub API URL and optionally a token for headless environments)
- List managed clusters from the hub
- Execute any `oc` command against a target cluster (pass-through args)
- Per-cluster token caching (OS keyring if available, otherwise `~/.config/multi-oc/tokens/<cluster>.token`)
- Discovery cache with TTL (default 60s, configurable)
- Airgap-friendly (vendored modules and prebuilt static Linux binary)

## Requirements
- Linux (RHEL/EL 8/9 recommended)
- `oc` CLI available in `PATH`
- RBAC on the hub to read `ManagedCluster`
- Direct HTTPS access from the jump host to the hub API and all managed cluster APIs
- Go (>= 1.21) only needed if you build from source

## Install
### From source
```bash
# Build (Go >= 1.21)
cd /path/to/multi-oc
go build -o moc .

# Optional: install system-wide
sudo install -m 0755 moc /usr/local/bin/moc
```

### Airgap (prebuilt binary)
```bash
# On an online workstation, produce or download the archive:
# moc-airgap-0.1.1.tar.gz

# On the jump host:
tar -xzf moc-airgap-0.1.1.tar.gz
sudo install -m 0755 moc-linux-amd64 /usr/local/bin/moc
moc version
```

## Quick start
```bash
# 1) Login to the hub (interactive prompt for URL; browser SSO by default)
moc login
# or non-interactive examples:
# moc login --hub https://api.hub.example:6443
# moc login --hub https://api.hub.example:6443 --headless --token 'sha256~...'

# 2) List clusters from the hub (cached for 60s by default)
moc ls

# 3) Run an oc command against a target cluster
moc <cluster-name> get nodes
moc <cluster-name> get ns -A
```

## Headless environments (no browser available)
- Hub login:
  - `moc login --headless` prompts for the hub API token (paste `sha256~...`).
- Target cluster execution:
  - If no token is cached for the target cluster, `moc` prints a token request URL of the form:
    - `https://oauth-openshift.apps.<cluster-domain>/oauth/token/request`
    - Open the URL from any machine with access, sign in, copy the token, paste it when prompted.
  - The prompt accepts the bare token (`sha256~...`) or full lines like `--token=sha256~...` or `oc login --token=...`.

### Environment variables (optional)
- `MOC_TARGET_TOKEN`:
  - If set, used as the target cluster token for the current call.
- `MOC_TARGET_CA_FILE`:
  - Path to a PEM CA bundle for the target cluster (if not provided by the hub).
- `MOC_TARGET_INSECURE=true`:
  - Skip TLS verification for the target cluster (only use if necessary).
- `MOC_DISCOVERY_TTL_SECONDS`:
  - Cache TTL for hub discovery (default `60`).

## Configuration, cache and token storage
- Hub URL: `~/.config/multi-oc/state.json`
- Discovery cache: `~/.config/multi-oc/cache/managedclusters.json` (respects `MOC_DISCOVERY_TTL_SECONDS`)
- Per-cluster tokens:
  - OS keyring (preferred), or
  - `~/.config/multi-oc/tokens/<cluster>.token` (0600)

## Security
- No persistent kubeconfigs for managed clusters are written.
- Tokens are cached per cluster in the OS keyring if available, otherwise as restricted files.
- Hub and target-cluster access always runs under your own user/SSO context.

## Troubleshooting
- Browser cannot be opened on the jump host:
  - Use headless login: `moc login --headless` and paste the token.
- Certificate issues to target cluster:
  - Provide a CA file: `export MOC_TARGET_CA_FILE=/path/to/ca.crt`
  - Or temporarily skip verify: `export MOC_TARGET_INSECURE=true` (not recommended long-term).
- Slow discovery:
  - Increase TTL: `export MOC_DISCOVERY_TTL_SECONDS=300`

## Credits
- Thorsten Stremetzne, People Visions & Magic LLP

## License
- TBD
