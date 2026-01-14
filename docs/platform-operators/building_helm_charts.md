# Building and Publishing a8s Helm Charts

This document describes how to build and publish a8s Helm charts to the public S3 bucket.

## Published Charts

The following a8s Helm charts are published to the public S3 bucket:

| Chart | S3 Path | Version |
|-------|---------|----------|
| PostgreSQL Operator | `anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/postgresql-operator/postgresql-operator-0.1.0.tgz` | 0.1.0 |
| Backup Manager | `anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/backup-manager/backup-manager-0.1.0.tgz` | 0.1.0 |
| Service Binding Controller | `anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/service-binding-controller/service-binding-controller-0.1.0.tgz` | 0.1.0 |

## Prerequisites

To build and push Helm charts, you'll need:

1. **Helm 3.0+** - Install from [helm.sh](https://helm.sh/docs/intro/install/)
2. **AWS CLI** - Install from [aws.amazon.com](https://aws.amazon.com/cli/)
3. **AWS S3 credentials** - Configure with `aws configure` for S3 uploads

Verify your setup:

```bash
helm version
aws --version
aws sts get-caller-identity  # Verify AWS credentials
```

## Building Charts Locally

### Package a Helm Chart

To package a chart locally (without pushing):

```bash
# Package the PostgreSQL Operator chart
helm package ./deploy/a8s/charts/a8s-postgresql-operator

# Package the Backup Manager chart
helm package ./deploy/a8s/charts/a8s-backup-manager

# Package the Service Binding Controller chart
helm package ./deploy/a8s/charts/a8s-service-binding-controller
```

This creates `.tgz` files (e.g., `postgresql-operator-0.1.0.tgz`) in your current directory.

### Test a Chart

To validate a chart before publishing:

```bash
# Lint a chart
helm lint ./deploy/a8s/charts/a8s-postgresql-operator

# Dry-run to see what would be installed
helm install my-release ./deploy/a8s/charts/a8s-postgresql-operator \
  --namespace a8s-system \
  --create-namespace \
  --dry-run --debug
```

## Publishing Charts to S3

### Step 1: Package Charts

Package your Helm charts in the `/tmp` directory:

```bash

# Package PostgreSQL Operator chart
mkdir ./tmp/postgresql-operator
helm package ./deploy/a8s/charts/a8s-postgresql-operator -d ./tmp/postgresql-operator

# Package Backup Manager chart
mkdir ./tmp/backup-manager
helm package ./deploy/a8s/charts/a8s-backup-manager -d ./tmp/backup-manager

# Package Service Binding Controller chart
mkdir ./tmp/service-binding-controller
helm package ./deploy/a8s/charts/a8s-service-binding-controller -d ./tmp/service-binding-controller
```

### Step 2: Upload to S3

```bash
# Upload PostgreSQL Operator chart to its folder
aws s3 cp ./tmp/postgresql-operator/postgresql-operator-0.1.0.tgz \
  s3://anynines-artifacts/charts/postgresql-operator/postgresql-operator-0.1.0.tgz

# Upload Backup Manager chart to its folder
aws s3 cp ./tmp/backup-manager/backup-manager-0.1.0.tgz \
  s3://anynines-artifacts/charts/backup-manager/backup-manager-0.1.0.tgz

# Upload Service Binding Controller chart to its folder
aws s3 cp ./tmp/service-binding-controller/service-binding-controller-0.1.0.tgz \
  s3://anynines-artifacts/charts/service-binding-controller/service-binding-controller-0.1.0.tgz
```

### Step 3: Generate and Upload Helm Repository Index Files

For each chart folder, first download existing charts from S3, then create an updated `index.yaml` file:

```bash
# Process PostgreSQL Operator chart
mkdir -p ./tmp/postgresql-operator
aws s3 sync s3://anynines-artifacts/charts/postgresql-operator/ ./tmp/postgresql-operator/
helm repo index ./tmp/postgresql-operator/ --url https://anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/postgresql-operator
aws s3 cp ./tmp/postgresql-operator/index.yaml \
  s3://anynines-artifacts/charts/postgresql-operator/index.yaml

# Process Backup Manager chart
mkdir -p ./tmp/backup-manager
aws s3 sync s3://anynines-artifacts/charts/backup-manager/ ./tmp/backup-manager/
helm repo index ./tmp/backup-manager/ --url https://anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/backup-manager
aws s3 cp ./tmp/backup-manager/index.yaml \
  s3://anynines-artifacts/charts/backup-manager/index.yaml

# Process Service Binding Controller chart
mkdir -p ./tmp/service-binding-controller
aws s3 sync s3://anynines-artifacts/charts/service-binding-controller/ ./tmp/service-binding-controller/
helm repo index ./tmp/service-binding-controller/ --url https://anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/service-binding-controller
aws s3 cp ./tmp/service-binding-controller/index.yaml \
  s3://anynines-artifacts/charts/service-binding-controller/index.yaml
```

**Note: Make sure to set make public using ACL on S3 console for all the repos updated**

## Using Published Charts

### Add Helm Repositories

Add the a8s Helm repositories to your Helm configuration:

```bash
# Add PostgreSQL Operator repository
helm repo add postgresql-operator \
  https://anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/postgresql-operator

# Add Backup Manager repository
helm repo add backup-manager \
  https://anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/backup-manager

# Add Service Binding Controller repository
helm repo add service-binding-controller \
  https://anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/service-binding-controller

# Update Helm repositories
helm repo update
```

### Install Charts Directly

```bash
# Install PostgreSQL Operator
helm install postgresql-operator \
  postgresql-operator/postgresql-operator \
  --namespace a8s-system \
  --create-namespace

# Install Backup Manager
helm install backup-manager \
  backup-manager/backup-manager \
  --namespace a8s-system

# Install Service Binding Controller
helm install service-binding-controller \
  service-binding-controller/service-binding-controller \
  --namespace a8s-system
```

## Troubleshooting

### "Access Denied" error when uploading to S3

```bash
# Verify your AWS credentials are configured
aws sts get-caller-identity

# Ensure your AWS user/role has S3 permissions for the anynines-artifacts bucket
```

### "Chart already exists" error

When uploading a chart version that already exists, you have two options:

1. **Update the version** in `Chart.yaml` and upload as a new version
2. **Force overwrite** the existing chart:

   ```bash
   aws s3 cp /tmp/a8s-postgresql-operator-0.1.0.tgz \
     s3://anynines-artifacts/charts/postgresql-operator-0.1.0.tgz \
     --metadata-directive REPLACE
   ```

### Helm repo add fails with HTTPS URL

If you encounter errors adding repositories, ensure:

1. The HTTPS URL is correct and accessible:

   ```bash
   curl https://anynines-artifacts.s3.eu-central-1.amazonaws.com/charts/postgresql-operator/index.yaml
   ```

2. The `index.yaml` file exists in the S3 bucket:

   ```bash
   aws s3 ls s3://anynines-artifacts/charts/postgresql-operator/
   # Should show index.yaml in the output
   ```

### "index.yaml not found" error

This occurs when Helm can't find the repository index file. Verify it was uploaded:

```bash
# Check if index.yaml exists in the S3 folder
aws s3 ls s3://anynines-artifacts/charts/postgresql-operator/

# Should show index.yaml and the chart .tgz files
```

### Cannot find chart after adding repository

Update your Helm repositories:

```bash
helm repo update
helm search repo a8s-postgresql-operator
```

## Additional Resources

- [Helm Documentation](https://helm.sh/docs/)
- [AWS S3 Documentation](https://docs.aws.amazon.com/s3/)
- [AWS CLI S3 Commands](https://docs.aws.amazon.com/cli/latest/userguide/cli-services-s3.html)
- [Chart.yaml Reference](https://helm.sh/docs/topics/charts/#the-chartyaml-file)
