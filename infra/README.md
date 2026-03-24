# Infrastructure and Deployment

This directory documents and manages the AWS runtime for `vimcolorschemes/worker`.

## Cloud runtime and scheduling

The worker runs in AWS as one-off ECS Fargate tasks triggered by EventBridge cron rules.

### How it is triggered

- `import`: `cron(0 13 * * ? *)`
- `update`: `cron(30 13 * * ? *)`
- `generate`: `cron(0 14 * * ? *)`
- `publish`: `cron(30 14 * * ? *)`

All schedules are in UTC.

### How jobs execute

- EventBridge rules run ECS tasks directly.
- Launch type is Fargate.
- The worker ECS service stays at desired count `0`; jobs run from schedules, not a long-running service.
- `import` uses the task default command.
- `update` overrides the command to `update`.
- `generate` overrides the command to `generate`.
- `publish` overrides the command to `publish`.

## CI deployment

`.github/workflows/deploy.yml` deploys on pushes to `main` and on manual dispatch.

Deployment behavior:

- Pushes two ECR tags: `${GITHUB_SHA}` and `latest`
- Registers a new ECS task definition revision in the `run-job` family
- Pins the ECS container image to the pushed image digest (`@sha256:...`)
- Ensures the ECS container maps `PUBLISH_WEBHOOK_URL` from Secrets Manager
- Updates EventBridge rules (`import`, `update`, `generate`, `publish`) to the new revision

Required GitHub Actions repo variables:

- `AWS_REGION`
- `AWS_REGISTRY_ID`
- `AWS_ROLE_TO_ASSUME`

The assumed role must trust GitHub OIDC (`token.actions.githubusercontent.com`) and allow:

- ECR push/read for `vimcolorschemes/worker`
- ECS task definition read/register (`DescribeTaskDefinition`, `RegisterTaskDefinition`)
- EventBridge target read/update (`ListTargetsByRule`, `PutTargets`)
- `iam:PassRole` for `ecsTaskExecutionRole` and `ecsEventsRole`

## Runtime secrets

The ECS task definition should inject these environment variables from AWS Secrets Manager:

- `GITHUB_TOKEN` from `vimcolorschemes/worker/github_token`
- `DATABASE_URL` from `vimcolorschemes/worker/database_url`
- `DATABASE_AUTH_TOKEN` from `vimcolorschemes/worker/database_auth_token`
- `PUBLISH_WEBHOOK_URL` from `vimcolorschemes/worker/publish_webhook_url`

`DATABASE_URL` is required by the worker and should point to a `libsql://...` endpoint in production.
For production deploys, keep all four values in ECS task `secrets` (not plain `environment`).

## Terraform

Terraform modules and run instructions are in `infra/terraform/README.md`.
