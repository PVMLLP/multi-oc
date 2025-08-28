# multi-oc (moc)

Zentrales CLI für OpenShift Multi-Cluster-Umgebungen (ohne ACM-Proxy), mit SSO-Login pro Cluster. Ideal für RHEL-Jumphosts mit direkter Netzwerkverbindung zu Hub und Managed Clustern.

## Voraussetzungen
- RHEL/EL 8/9 (oder beliebiges Linux)
- `oc` CLI im `PATH`
- Zugriff auf Hub-Cluster (RBAC: Lesen von ManagedCluster)
- Direkter HTTPS-Zugriff auf die API-Server der Ziel-Cluster

## Installation
```bash
# Build (Go >= 1.21)
cd /path/to/multi-oc
go build -o moc .

# Optional: nach /usr/local/bin
sudo install -m 0755 moc /usr/local/bin/moc
```

## Nutzung
```bash
# 1) Am Hub einloggen (Browser-SSO)
moc login --hub https://api.hub.example:6443

# 2) Cluster vom Hub auflisten
moc ls

# 3) Ein oc-Kommando auf einem Ziel-Cluster ausführen (Browser-SSO für Zielcluster)
moc <cluster-name> get nodes
moc <cluster-name> get ns -A
```

Hinweise:
- `moc ls` liest `ManagedCluster` am Hub und extrahiert pro Cluster die API-URL (und CA, sofern vorhanden).
- Für jeden Ziel-Cluster-Aufruf erzeugt `moc` eine temporäre Kubeconfig unter `/tmp` und ruft `oc` mit `KUBECONFIG` auf. Die Datei wird im Anschluss gelöscht.
- TLS: Wenn der Hub die `caBundle` bereitstellt, wird diese genutzt. Fehlt sie, nutzt `oc login` `--insecure-skip-tls-verify=true`.

## Sicherheit / Compliance
- Keine dauerhafte Ablage von Managed-Cluster-Kubeconfigs.
- Login via OpenShift OAuth (Browser-Flow/SSO, i. d. R. AD/IdP-Integration).
- Tokens verbleiben im Kontext der temporären Kubeconfigs und werden nach Nutzung verworfen.

## Bekannte Einschränkungen (MVP)
- `moc ls` setzt voraus, dass `oc` aktuell gegen den Hub eingeloggt ist (wird durch `moc login --hub` erreicht).
- Keychain-Speicher für Tokens ist vorbereitet, aber derzeit nicht aktiv im Einsatz.
- Autocomplete/Fuzzy-Matching folgt in einem späteren Schritt.

## Support
Issues und Ideen willkommen.
