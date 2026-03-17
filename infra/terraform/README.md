# Terraform Setup

This directory manages static AWS infrastructure for `vimcolorschemes/worker`.

## Layout

- `bootstrap/`: creates the remote state backend (S3 + DynamoDB lock table)
- `envs/prod.tfvars.example`: production variable template
- root module: ECS/EventBridge/ECR/IAM/Secrets metadata resources

## Bootstrap remote state

Run from `infra/terraform/bootstrap`:

```shell
tofu init
tofu apply -var='aws_profile=YOUR_AWS_PROFILE'
```

## Initialize main module

Run from `infra/terraform`:

```shell
cp envs/prod.tfvars.example envs/prod.tfvars
cp backend.hcl.example backend.hcl
tofu init -backend-config=backend.hcl.example
```

Then fill local values in `envs/prod.tfvars` and `backend.hcl`, and run:

```shell
tofu init -backend-config=backend.hcl -reconfigure
```

## Scope

This module is configured to let CI keep owning deploy-time task definition revision updates.

- EventBridge target `task_definition_arn` changes are ignored in Terraform.
- Secrets are modeled as `aws_secretsmanager_secret` only (secret values are not managed here).
