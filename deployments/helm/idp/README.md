# OpenShift Deployment Runbook

This directory contains the Helm umbrella chart used to deploy and manage the **Thunder Identity Management Service** on Red Hat OpenShift across all environments:
* **Development:** `nsw-infra-dev`
* **Staging:** `nsw-infra-staging` (Manual GitOps overrides)
* **Production:** `nsw-infra-prod` (Manual baseline)

---

## Directory Structure

```
deployments/helm/idp/
├── Chart.yaml              # Parent umbrella chart metadata and local aliases
├── Chart.lock              # Locked dependency versions and digests
├── Dockerfile              # Builds the secure custom runner image mapping for IDP
├── values.yaml             # Default umbrella settings (Hardened for OpenShift)
├── templates/
│   ├── route.yaml          # Native OpenShift Route resource (exposes public routes)
│   └── post-install-seed.yaml # Hook to trigger sample resource provisioning
├── files/
│   └── 02-sample-resources.sh # Seeding post-install hook shell script
└── charts/
    └── thunderid/          # Persistent, uncompressed local sub-chart (Hardened)
```

---

## OpenShift Hardening & Customizations

The Identity Provider is configured to comply with OpenShift Enterprise standards. The following modifications are maintained within the `charts/thunderid/` local directory:

1. **SCC Compliance (Rootless)**:
   * **Modification:** Hardcoded `runAsUser` values are managed via `.Values.deployment.securityContext`. For OpenShift, we ensure the namespace SCC (restricted-v2) can dynamically assign UIDs.
   * **Note:** `readOnlyRootFilesystem` is set to `false` in `values.yaml` to allow the bootstrap setup-job to initialize required binaries.
2. **Port Realignment (Port 8080)**:
   * The system is configured to use port **`8080`** for both internal container bindings and external service access to match standard ingress patterns.
3. **OAuth Redirect URI Sensitivity**:
   * **CRITICAL:** The Asgardeo/Thunder SDK is highly sensitive to trailing slashes. All registered applications MUST include both variations:
     - `https://<domain>/console`
     - `https://<domain>/console/`
   * This is handled automatically by the `02-sample-resources.sh` seeding script.

---

## Managing the ThunderID Sub-Chart

Unlike standard Helm charts that pull from remote registries, this repo maintains a **persistent local sub-chart** in `charts/thunderid/`. This allows us to track hardening changes (like SCC patches) directly in Git.

### To Upgrade the Upstream Version:
1. Download the new upstream tarball.
2. Extract it and replace the contents of `deployments/helm/idp/charts/thunderid/`.
3. Re-apply any specific hardening patches to the templates.
4. Update the `version` in `deployments/helm/idp/Chart.yaml`.
5. Synchronize the lock file:
   ```bash
   helm dependency update deployments/helm/idp
   ```

---

## Deployment Execution Guide (Manual Sync)

Because WSO2 infrastructure components may bypass automatic GitOps synchronization in some environments, use the following manual deployment steps:

### 1. Synchronize Dependencies
Ensure the local `Chart.lock` is up to date:
```bash
helm dependency build deployments/helm/idp
```

### 2. Retrieve Seeding Credentials
Extract the required passwords from the cluster:
```bash
export ADMIN_PASS=$(oc get secret nsw-db-credentials -n nsw-infra-staging -o jsonpath='{.data.password}' | base64 -d)
export SAMPLE_PASS="postgres"
export M2M_SECRET="postgres"
```

### 3. Run the Helm Upgrade
Execute the manual upgrade:
```bash
helm upgrade --install idp-thunder deployments/helm/idp \
  --namespace nsw-infra-staging \
  -f deployments/helm/idp/values.yaml \
  --set seeding.adminPassword="$ADMIN_PASS" \
  --set seeding.sampleUserPassword="$SAMPLE_PASS" \
  --set seeding.m2mClientSecret="$M2M_SECRET" \
  --wait
```

### 4. Confirm Healthiness
```bash
# Check running pod statuses
oc get pods -l app.kubernetes.io/name=thunder

# Smoke test the public Route console (Note the trailing slash!)
curl -kI "https://idp-thunder-nsw-infra-staging.apps.sovecloud1.akaza.lk/console/"
```
