# GCP Artifact Registry Docker Images Cleaner

## Objective

The objective is to delete on `GCP Artifacts Registry` docker images that:

- have no tag
- or that are older than a given number of days and don't have tags containing `latest`

## args

- project (mandatory): GCP Project ID
- location (mandatory): location of GCP Artifact Registry
- repository (mandatory): GCP Artifact Registry repository name
- days-to-keep (mandatory): number of days used to filter docker image on build date
- dry (optional, default `false`): if set, will only count and display the number of tags and images to delete.

## Run on GKE

to run the command in GKE, build and push the image (see `Dockerfile`), create a GCP service account (and grant permission, see `k8s/gcp_service_account` folder) and deploy it on gke as a cronjob (see `k8s` folder)
