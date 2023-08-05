# GCP Artifact Registry Docker Images Cleaner

## Objective

The objective is to delete on `GCP Artifacts Registry` docker images that:

- have no tag 
- or that are older than a given number of days and don't have tags containing `latest`

## args

- project (mandatory): GCP Project ID
- repository (mandatory): GCP Artifact Registry repository name
- daysToKeep (mandatory): number of days used to filter docker image on build date
- dry (optional, default `false`): if set, will only count and display the number of tags and images to delete.