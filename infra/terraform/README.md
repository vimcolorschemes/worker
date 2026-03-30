# Terraform Setup

For runtime/deploy behavior and IAM expectations, see `infra/README.md`.

This directory manages static AWS infrastructure for `vimcolorschemes/worker`.

## Layout

- `bootstrap/`: creates the remote state backend (S3 + DynamoDB lock table)
- `envs/prod.tfvars.example`: production variable template
- root module: ECS/EventBridge/ECR/IAM/Secrets metadata resources, including SNS email notifications

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
- Runtime secret names expected by the worker are:
  - `vimcolorschemes/worker/github_token`
  - `vimcolorschemes/worker/database_url`
  - `vimcolorschemes/worker/database_auth_token`
- ECS task definitions should map these as container `secrets`.
- `alert_email_addresses` controls SNS email subscriptions for job notifications.
- `JOB_NOTIFICATIONS_TOPIC_ARN` and `PUBLISH_WEBHOOK_URL` are treated as non-secret and should be set as plain container `environment` during deploy.
- After `tofu apply`, confirm the SNS subscription emails before expecting notifications.
