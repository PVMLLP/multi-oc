## multi-oc (moc) — Enterprise Operations Guide

### Document purpose
This document provides an enterprise-grade, compliance-aware operations guide for the “multi-oc” (moc) command-line tool. It is intended for Operations, SRE, Platform, and Security teams who run OpenShift multi-cluster environments and need a standardized, auditable way to interact with managed clusters from jump hosts. This guide assumes distribution as a compiled artifact (`moc-airgap-<version>.tar.gz`) without source code access.

---

## What is moc?

moc is a centralized CLI that streamlines OpenShift multi-cluster operations from a single shell. It:

- Uses a Hub cluster as the source of truth to discover managed clusters.
- Automates login flows for the Hub and target clusters.
- Optimizes for restricted environments (e.g., air-gapped jump hosts without local browsers).
- Minimizes operational friction by caching tokens and optionally consuming per-cluster kubeconfigs.

Typical use case: An engineer connects to a hardened Linux jump host (RHEL/EL), uses moc to log into the Hub and then runs `oc` commands against any managed cluster without manually switching kubeconfigs or contexts.

---

## Why do we need it?

- Reduce operational toil: No manual kubeconfig juggling or ad-hoc token lookups.
- Improve speed-to-triage: Autodiscovery of clusters and auto-handling of login flows.
- Hardened workflows: Works reliably in headless/air-gapped environments common in enterprise.
- Standardization: Enforce consistent patterns for authentication, secrets handling, and audited interactions.

---

## Key capabilities

- Headless Hub login: Generates a copy-paste token request URL for the Hub, prompts once, and logs in using `oc login --token`.
- Automatic token request for target clusters: On first interaction with a target cluster, moc prints a copyable OAuth URL and prompts for token, then caches it.
- Token caching: Per-cluster tokens cached via OS keyring (with secure file fallback).
- Kubeconfig support: If available, moc will prefer per-cluster kubeconfigs over tokens. Kubeconfigs can be:
  - Placed by Ops under a well-known path.
  - Imported in bulk from the Hub using `moc kubeconfigs`.
  - Optionally generated from ManagedServiceAccount (MSA) token secrets on the Hub.
- Fail-safe retries: When an existing token fails (e.g., expired), moc automatically discards it and prompts for a fresh one.

---

## What problems does moc solve?

- Headless authentication in clean RHEL environments (no local browser).
- Centralized, reproducible cluster discovery and access.
- Consistent operator workflow across teams, environments, and network segments.
- Reduced support tickets caused by misconfigured kubeconfig contexts or stale tokens.

---

## Risks and considerations

- Credential handling:
  - Tokens are cached (keyring or secure file fallback). Enforce least privilege and rotation.
  - Ensure file permissions (0600) on fallback token and kubeconfig files.
- RBAC boundaries:
  - Hub login does not imply target cluster access. Users need proper permissions on target clusters.
- Availability coupling:
  - Discovery and some automations depend on Hub access. Hub outages impact those operations.
- Compliance:
  - Treat tokens as secrets (no log leakage, safe copy-paste practices).
  - Review retention and backup scope for `~/.config/multi-oc`.
- Operational prerequisites:
  - `oc` CLI must be installed and on PATH.
  - Network access to Hub API and to target cluster APIs (unless using a Hub proxy model).

---

## Architecture and components

- moc binary (Go-based CLI) orchestrates:
  - Hub discovery via `oc get managedclusters.cluster.open-cluster-management.io -o json`.
  - Headless login flows that emit “copy-paste” OAuth URLs.
  - Execution of `oc` against target clusters with proper auth flags.
- Token caching:
  - Primary: OS keyring (preferred).
  - Fallback: `~/.config/multi-oc/tokens/<cluster>.token` (0600).
- Kubeconfig preference:
  - Env override via `MOC_TARGET_KUBECONFIG`.
  - Convention path: `~/.config/multi-oc/kubeconfigs/<cluster>.kubeconfig`.
  - Optional bulk import or on-demand retrieval via Hub (when available).

---

## Supported platform and dependencies

- Operating Systems:
  - RHEL/EL 8/9 (primary target), Linux in general.
  - macOS supported (for development/testing).
- Dependencies:
  - `oc` CLI available in PATH.
  - Go-built static binary (no runtime Go required).
- Optional Hub-side capabilities:
  - ManagedServiceAccount (MSA) add-on for token-based kubeconfig construction (optional).
  - Hive/ClusterDeployment labels (if available) for alternative kubeconfig discovery.

---

## Installation (air-gapped)

1. Transfer the release archive to the jump host:
   - `moc-airgap-<version>.tar.gz`
2. Verify integrity (checksum/signature if your org enforces it).
3. Extract the archive:
   ```bash
   tar -xzf moc-airgap-<version>.tar.gz
   ```
4. Install binary to a standard location (optional):
   ```bash
   sudo install -m 0755 moc-linux-amd64 /usr/local/bin/moc
   ```
   Alternatively, run it in-place as `./moc-linux-amd64`.

---

## Configuration

- Hub state file:
  - `~/.config/multi-oc/state.json` (stores Hub API URL).
- Token cache:
  - Keyring (preferred).
  - Fallback: `~/.config/multi-oc/tokens/<cluster>.token` (0600).
- Kubeconfigs:
  - `~/.config/multi-oc/kubeconfigs/<cluster>.kubeconfig`.
- Cache for discovery (TTL):
  - `~/.config/multi-oc/cache/managedclusters.json` (short TTL, default ~60s).

### Environment variables

| Variable                   | Scope   | Purpose                                                  |
|---------------------------|---------|----------------------------------------------------------|
| MOC_HUB_INSECURE          | Hub     | If “true”, use `--insecure-skip-tls-verify`             |
| MOC_HUB_CA_FILE           | Hub     | Path to Hub CA file                                      |
| MOC_TARGET_TOKEN          | Target  | Pre-supply a cluster token (avoids prompt)              |
| MOC_TARGET_INSECURE       | Target  | If “true”, use `--insecure-skip-tls-verify`             |
| MOC_TARGET_CA_FILE        | Target  | Path to target cluster CA file                           |
| MOC_TARGET_KUBECONFIG     | Target  | Path to per-cluster kubeconfig to override discovery    |
| MOC_DISCOVERY_TTL_SECONDS | Discov. | TTL for cached cluster list (default 60)                 |

---

## Usage (with automatic token request & caching)

The following workflow describes the headless pattern recommended for hardened jump hosts (no local browser).

### 1) Login to the Hub (headless)

```bash
moc login --hub https://api.<hub-domain>:6443
```

What happens:
- moc saves the Hub API URL to local state.
- moc prints a copy-paste OAuth Token Request URL:
  ```
  https://oauth-openshift.apps.<hub-base-domain>/oauth/token/request
  ```
- Open that URL from any machine with browser access, sign in, copy the token (`sha256~…`), and paste into the terminal prompt.
- moc runs `oc login --token` against the Hub and caches the session.

Notes:
- If already logged into the correct Hub, moc detects it and skips prompting.
- TLS options:
  - `--insecure` or `MOC_HUB_INSECURE=true`
  - `--ca-file /path/to/ca.crt` or `MOC_HUB_CA_FILE=/path/to/ca.crt`

### 2) List managed clusters

```bash
moc ls
```

Uses Hub to retrieve managed clusters and cache the list for a short TTL.

### 3) First-time interaction with a target cluster (auto token request)

```bash
moc <cluster-name> get nodes
```

Behavior:
- If a kubeconfig exists for the cluster, moc uses it directly (no token prompt).
- If no kubeconfig exists and no token is cached:
  - moc prints a copy-paste OAuth Token Request URL derived from the target cluster API:
    ```
    https://oauth-openshift.apps.<target-base-domain>/oauth/token/request
    ```
  - Paste the token (`sha256~…`) once; moc caches it securely (keyring or secure file fallback).
- Future invocations use the cached token automatically.
- If a 401 occurs (expired/invalid token), moc deletes the cached token, explains the failure, and prompts for a fresh token once.

### 4) Optional: Preload kubeconfigs for all clusters

If your Hub exposes kubeconfig material in any of the supported forms (e.g., via ManagedServiceAccount add-on), you can bulk import:

```bash
moc kubeconfigs --verbose --force
# Optional, if ManagedServiceAccount integration is in place:
moc kubeconfigs --verbose --msa-name moc
```

Sources checked (in order):
1) `admin-kubeconfig` secret in the cluster namespace  
2) `<cluster>-admin-kubeconfig` secret in the cluster namespace  
3) `managedcluster-info` secret in the cluster namespace (`kubeconfig` key)  
4) Hive-labeled secrets across all namespaces (`hive.openshift.io/cluster-deployment-name=<cluster>`, `kubeconfig` key)  
5) ManagedServiceAccount token secret(s) (e.g., `<msa>-token`) for building a kubeconfig on-the-fly  

Written to: `~/.config/multi-oc/kubeconfigs/<cluster>.kubeconfig`

If none of the above is available for a cluster, moc will skip it and fall back to the token prompt on first use.

---

## Operational runbook

### Day-1 (initial setup)
- Install `moc` on jump hosts.
- Ensure `oc` is installed and on PATH.
- Ensure network access to Hub and target clusters’ APIs.
- Login to Hub (`moc login`) and verify `moc ls`.

### Day-2 (ongoing operations)
- Use `moc <cluster> <oc-args>` for day-to-day tasks.
- If tokens expire, moc will prompt once and re-cache them.
- Optionally, preload kubeconfigs with `moc kubeconfigs` so no token prompts occur.

### Token rotation & cleanup
- Delete a cached token for a cluster by removing its secure entry (moc does this automatically on 401).
- File fallback location (only if keyring unavailable):
  - `~/.config/multi-oc/tokens/<cluster>.token` (ensure 0600 permissions).

### Upgrades
- Replace the binary from the latest `moc-airgap-<version>.tar.gz`.
- Backward compatibility: existing state and caches are preserved.

### Backups
- Typically not required. If you need to preserve state:
  - `~/.config/multi-oc/state.json` (Hub URL)
  - `~/.config/multi-oc/kubeconfigs/*`
- Avoid backing up token files unless strictly necessary; treat as secrets.

---

## Security & compliance

- Authentication:
  - Supports Hub and target clusters via OAuth tokens or kubeconfigs.
  - Headless flows print copy-paste URLs for token retrieval; no browser automation.
- Secrets management:
  - Tokens cached within OS keyring (preferred) or secure file fallback (0600).
  - Kubeconfigs written with 0600 permissions.
- Authorization:
  - Enforced by the OpenShift clusters; moc does not bypass RBAC.
- Auditability:
  - All `oc` interactions are standard and auditable in OpenShift API audit logs.
  - Recommend enabling Hub and target cluster audit logs per corporate policy.
- Least privilege:
  - If using MSA-generated kubeconfigs, scope managed permissions to least privilege per use case (e.g., read-only).
- Data at rest:
  - Keep `~/.config/multi-oc` outside of backups unless justified; if included, treat as sensitive.

---

## Troubleshooting

| Symptom / Message                                                                 | Likely Cause                                  | Resolution                                                                                         |
|-----------------------------------------------------------------------------------|-----------------------------------------------|----------------------------------------------------------------------------------------------------|
| “Please pass oc arguments, e.g.,: get nodes”                                      | Missing oc subcommand                         | Use `moc <cluster> <oc-args>` (e.g., `moc de08-lkd10 get nodes`)                                  |
| “You must be logged in to the server (…provide credentials)”                      | Missing/invalid token or kubeconfig           | On first use, moc prompts for token and caches it; on 401, moc re-prompts and re-caches            |
| “no kubeconfig secret found on hub …” during `moc kubeconfigs`                    | Hub does not expose kubeconfigs by default    | Use token flow per-cluster or enable MSA/Hive integration to expose kubeconfigs on the Hub         |
| Hub login keeps prompting                                                         | Hub session expired or wrong Hub URL          | Re-run `moc login`, verify Hub URL and TLS settings (`--insecure` or `--ca-file`)                 |
| “oc not found”                                                                    | `oc` CLI missing                              | Install `oc` and ensure it is in PATH                                                              |
| TLS errors                                                                        | Untrusted CA                                  | Provide `--ca-file` or set `MOC_HUB_CA_FILE` / `MOC_TARGET_CA_FILE`; avoid `--insecure` in prod    |

---

## Change management and versioning

- Releases are distributed as `moc-airgap-<version>.tar.gz`.
- Patch version auto-increments on builds; include release notes in your internal registry or artifact repository.
- Track changes in your CMDB/operational change records (link to ticket/approval).

---

## Support and ownership

- Primary owners: Platform/SRE team operating OpenShift and Hub.
- Security review: Ensure token handling and file permissions comply with internal policies.
- Escalation path: Platform → Security → Vendor (if using ACM/MSA add-ons) per your standard process.

---

## Appendix: Command reference

### Summary

| Command                                              | Description                                                           |
|------------------------------------------------------|-----------------------------------------------------------------------|
| `moc login --hub <https://api…:6443>`                | Headless login to the Hub (copy-paste URL, prompt for token)         |
| `moc ls`                                             | List managed clusters from the Hub                                    |
| `moc <cluster> <oc-args>`                            | Execute `oc` against a target cluster; auto token request & caching   |
| `moc kubeconfigs [--verbose] [--force] [--msa-name]` | Bulk import kubeconfigs when available on Hub                         |

### Examples

Headless Hub login:
```bash
moc login --hub https://api.hub.example:6443
```

List clusters:
```bash
moc ls
```

First interaction with a target cluster (auto token request and cache):
```bash
moc de08-lkd10 get nodes
```

Import kubeconfigs (verbose, overwrite existing):
```bash
moc kubeconfigs --verbose --force
# Optional, if ManagedServiceAccount integration is in place:
moc kubeconfigs --verbose --msa-name moc
```

---

## Closing note

moc is designed to work “with the grain” of enterprise constraints: headless jump hosts, no local browser, strong RBAC, and compliance-driven operations. Start with the headless token flow (automatic URL + cache), and consider kubeconfig preloading via MSA/Hive if your Hub supports it.


