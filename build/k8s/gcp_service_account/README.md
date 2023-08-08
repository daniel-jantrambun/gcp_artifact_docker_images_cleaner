# GCP Service Account

A GCP service account with correct permission is needed to be allowed to do action on GCP artifact Registry

## Instructions

- create a service_account `artifact-registry-cleaner` on GCP
- grant the role `Artifact Registry Repository Administrator` to this service account
- create a key for this service account and download the key
- create a kubernetes secret named `artifact-registry-cleaner-service-account`

### Create Service Account

```bash
gcloud iam service-accounts create artifact-registry-cleaner \
--description="for kubernetes job to clean artifact registry docker images" \
--display-name="artifact-registry-cleaner"
```

### Grant role

```bash
gcloud projects add-iam-policy-binding <PROJECT_ID> \
--member=serviceAccount:artifact-registry-cleaner@<PROJECT_ID>.iam.gserviceaccount.com \
--role=roles/artifactregistry.repoAdmin
```

### Create key

```bash
gcloud iam service-accounts keys create ~/artifact-registry-cleaner.json \
--iam-account=artifact-registry-cleaner@<PROJECT_ID>.iam.gserviceaccount.com
```

### create Kubernetes secret

```bash
kubectl create secret generic artifact-registry-cleaner-service-account --from-file=~/artifact-registry-cleaner.json --namespace=artifact-registry-cleaner --dry-run=client -o yaml | kubectl apply -f -
```
