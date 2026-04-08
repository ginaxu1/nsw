# NSW Platform — Helm Deployment Guide

> **Target:** OpenShift (Akaza GovCloud)  
> **Constraint:** Always pass `--history-max 1` to stay within the 20-secret quota limit for multiple OGA pods.

---

## Chart Inventory

| Release Name | Chart Source | Description |
|:---|:---|:---|
| `nsw-api` | `./deployments/helm/nsw-api` | Core backend API (Go) |
| `trader-app` | `./deployments/helm/trader-app` | Trader portal frontend (React) |
| `oga-<agency>-app` | `./deployments/helm/oga-app` | Generic OGA portal frontend (React) |
| `oga-<agency>-backend` | `./deployments/helm/oga-backend` | Generic OGA backend API (Go) |
| `idp-thunder` | `./deployments/helm/idp` | Declarative Identity Provider (WSO2) Umbrella Chart |
| `temporal` | `./deployments/helm/temporal` | Workflow Engine (Server + UI) |

---

## 1. Build & Push Images (GHCR)

Build with `linux/amd64` when on Apple Silicon:

```bash
# From the repository root
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/nsw-api:latest -f backend/Dockerfile ./backend --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/trader-app:latest -f portals/apps/trader-app/Dockerfile ./portals --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/oga-app:latest -f portals/apps/oga-app/Dockerfile ./portals --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/oga-backend:latest -f oga/Dockerfile ./oga --push
docker buildx build --platform linux/amd64 -t ghcr.io/opennsw/idp:latest -f deployments/helm/idp/Dockerfile ./deployments/helm/idp --push
```

---

## 2. Deploy / Upgrade Helm Charts (Multi-Environment)

Always explicitly execute Helm layering the environment values.

### Developer Initialization
```bash
helm dependency build ./deployments/helm/idp
helm dependency build ./deployments/helm/nsw-api
```

### Option A: STAGING Environment (national-single-window-platform)
```bash
# Core Services
helm upgrade --install staging-api ./deployments/helm/nsw-api -f ./deployments/helm/nsw-api/values.yaml -f ./deployments/helm/nsw-api/values-staging.yaml -n national-single-window-platform --history-max 1
helm upgrade --install staging-trader-app ./deployments/helm/trader-app -f ./deployments/helm/trader-app/values.yaml -f ./deployments/helm/trader-app/values-staging.yaml -n national-single-window-platform --set fullnameOverride=staging-trader-app --history-max 1

# OGA Apps & Backends (NPQS, FCAU, IRD)
helm upgrade --install staging-oga-fcau-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/values.yaml -f ./deployments/helm/oga-backend/fcau-backend-values.yaml -n national-single-window-platform --history-max 1
helm upgrade --install staging-oga-fcau ./deployments/helm/oga-app -f ./deployments/helm/oga-app/values.yaml -f ./deployments/helm/oga-app/values/fcau-values.yaml -n national-single-window-platform --set fullnameOverride=staging-oga-fcau --history-max 1
```

### Option B: DEV Environment (national-single-window-platform)
```bash
# Core Services
helm upgrade --install dev-nsw-api ./deployments/helm/nsw-api -f ./deployments/helm/nsw-api/values.yaml -f ./deployments/helm/nsw-api/values-dev.yaml -n national-single-window-platform --history-max 1
helm upgrade --install dev-trader-app ./deployments/helm/trader-app -f ./deployments/helm/trader-app/values.yaml -f ./deployments/helm/trader-app/values-dev.yaml -n national-single-window-platform --history-max 1

# OGA Apps & Backends (NPQS, FCAU, IRD)
helm upgrade --install dev-oga-fcau-backend ./deployments/helm/oga-backend -f ./deployments/helm/oga-backend/values.yaml -f ./deployments/helm/oga-backend/fcau-backend-values.yaml -n national-single-window-platform --history-max 1
helm upgrade --install dev-oga-fcau ./deployments/helm/oga-app -f ./deployments/helm/oga-app/values.yaml -f ./deployments/helm/oga-app/values-dev.yaml -n national-single-window-platform --set fullnameOverride=dev-oga-fcau --history-max 1
```

---

## 3. Database & Seeding

### 3.1 Initial Seeding
The `nsw-api` chart includes a `pre-install` hook that runs `init.sql`. This script is idempotent and seeds:
- `workflow_template_v2` (V2 Workflow definitions)
- `hs_codes`
- `customs_house_agents`
- `forms` (schemas and UI templates)

### 3.2 Troubleshooting "Record Not Found"
If specific tasks like `node_2_cusdec` are reported as "not found", ensure:
1.  **Fresh Consignment**: Create a new consignment in the Trader portal. Old IDs after a DB purge will fail.
2.  **Seeding Verification**: Check `workflow_template_maps_v2` for the correct HS code mapping.
3.  **Temporal Workers**: Ensure `dev-nsw-api` pods are healthy and connected to Temporal.

---

## 4. Diagnostic Commands
```bash
# Check OGA Connectivity
oc exec deployment/dev-nsw-api -- curl -v http://dev-oga-npqs-backend:8081/health

# Verify Database Seeding
oc exec deployment/nsw-db -- psql -U postgres -d nsw_dev -c "SELECT id FROM workflow_template_v2;"
```