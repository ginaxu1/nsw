# WSO2 ThunderID Helm Chart — OpenShift Deployment Runbook

This directory contains the Helm umbrella chart used to deploy and manage the **WSO2 ThunderID Identity Management Service** on Red Hat OpenShift across all environments:
* **Development:** `nsw-infra-dev`
* **Staging:** `nsw-infra-staging` (Manual GitOps overrides)
* **Production:** `nsw-infra-prod` (Manual baseline)

---

## Directory Structure

```
deployments/helm/idp/
├── Chart.yaml              # Parent umbrella chart metadata and local aliases
├── Dockerfile              # Builds the secure custom runner image mapping for IDP
├── values.yaml             # Default umbrella settings
├── templates/
│   └── route.yaml          # Native OpenShift Route resource (exposes public routes)
├── files/
│   └── 02-sample-resources.sh # Seeding post-install hook shell script
└── charts/
    └── thunderid-0.37.0.tgz # Patched, compressed binary sub-chart archive
```

---

## OpenShift Hardening Policies (Required Patches)

The upstream WSO2 ThunderID Helm chart is incompatible with native OpenShift clusters out-of-the-box. Before packaging any new release, the templates **must** be patched with the following enterprise constraints:

1. **Security Context Constraints (SCC) Rootless Compliance**:
   * Upstream templates hardcode static container UIDs (`runAsUser: 10001`). 
   * **Patch:** Strip all hardcoded `runAsUser` or `runAsGroup` keys from the container and pod `securityContext` sections. This allows OpenShift's strict `restricted-v2` SCC profile to dynamically assign randomized, secure UIDs to each container.
2. **Port Realignment (Port 8080)**:
   * Upstream templates configure health checks and container bindings to port `8090`.
   * **Patch:** Realign all service bindings, container ports, readiness probes, and liveness probes back to the standard port **`8080`** to match enterprise ingress routes.
3. **SubPath Secret Mounts**:
   * **Patch:** Ensure custom signing/encryption keys (`signing.key` and `crypto.key`) are mounted inside `/opt/thunderid/secrets/` using explicit, isolated `subPath` bindings. This prevents directory-overwrite collisions inside the container runtime.

---

## Upgrading WSO2 Thunder to a New Upstream Release

When WSO2 releases a new version of Thunder (e.g. `v0.38.0`), follow this step-by-step process to update the local packages in this repository:

### Step 1: Download & Extract the Upstream Chart
Obtain the official Helm chart from the upstream [Asgardeo Thunder Repository](https://github.com/asgardeo/thunder/tree/main/install/openchoreo/helm/charts/thunderid-component) and extract it locally:
```bash
# Example: Extracting the new upstream component chart
tar -xzf thunderid-0.38.0.tgz -C /tmp/uncompressed-thunder/
```

### Step 2: Apply the OpenShift-Hardening Patches
Apply the three hardening patches detailed in the section above to the templates inside `/tmp/uncompressed-thunder/`.

### Step 3: Package the Patched Chart
Package the customized directory back into a compressed tarball and place it inside the `charts/` subdirectory of this repository:
```bash
# Package the chart
helm package /tmp/uncompressed-thunder/ --destination deployments/helm/idp/charts/
```
*Note: This generates `deployments/helm/idp/charts/thunderid-0.38.0.tgz`. Clean up the uncompressed `/tmp/` directory after packaging.*

### Step 4: Update Dependency Declarations
Edit `deployments/helm/idp/Chart.yaml` to point to the new version:
```yaml
dependencies:
  - name: thunderid
    version: 0.38.0             # <-- Update to your new packaged version
    repository: "file://./charts"
    alias: thunder
```

### Step 5: Validate the Assembly
Run a local rendering test to verify that the tarball resolves and compiles successfully without errors:
```bash
helm template idp-test deployments/helm/idp -f ../../../nsw-gitops/envs/base/infra-umbrella/charts/idp-thunder/values-staging.yaml
```

---

## Deployment Execution Guide (Manual Sync)

Because WSO2 infrastructure components bypass automatic GitOps synchronization, deployments must be triggered manually using `helm` and the `oc` CLI.

### 1. Login to OpenShift
Ensure your active shell is logged into the target cluster and pointing to the target namespace:
```bash
oc project nsw-infra-staging
```

### 2. Retrieve Database & Seeding Passwords
To allow the post-install seeder to connect and seed application databases successfully, extract the active passwords from the target cluster's credentials secret:
```bash
export ADMIN_PASS="$(oc get secret idp-thunder-seeder-credentials -o jsonpath='{.data.adminPassword}' | base64 -d)"
export SAMPLE_PASS="$(oc get secret idp-thunder-seeder-credentials -o jsonpath='{.data.sampleUserPassword}' | base64 -d)"
export M2M_SECRET="$(oc get secret idp-thunder-seeder-credentials -o jsonpath='{.data.m2mClientSecret}' | base64 -d)"
```

### 3. Run the Helm Upgrade
Execute the manual upgrade inside the target namespace:
```bash
helm upgrade --install idp-thunder deployments/helm/idp \
  --namespace nsw-infra-staging \
  -f ../../../nsw-gitops/envs/base/infra-umbrella/charts/idp-thunder/values-staging.yaml \
  --set seeding.adminPassword="$ADMIN_PASS" \
  --set seeding.sampleUserPassword="$SAMPLE_PASS" \
  --set seeding.m2mClientSecret="$M2M_SECRET" \
  --wait=false
```

### 4. Trigger Configuration Rolling Restarts
To force the main server pods to pick up updated configuration maps instantly, trigger a rolling rollout:
```bash
oc rollout restart deployment/idp-thunder-deployment
oc rollout status deployment/idp-thunder-deployment
```

### 5. Confirm Healthiness
Verify that all pods, databases, and routes are 100% active and healthy:
```bash
# Check running pod statuses
oc get pods -l app.kubernetes.io/name=thunder

# Smoke test the public Route console
curl -kI "https://idp-thunder-nsw-infra-staging.apps.sovecloud1.akaza.lk/console/"
```
