# Infrastructure and Deployment

This directory documents and manages the AWS runtime for `vimcolorschemes/worker`.

## Cloud runtime and scheduling

The worker runs in AWS as one-off ECS Fargate tasks triggered by EventBridge cron rules.

### How it is triggered

- `import`: `cron(0 13 * * ? *)`
- `update`: `cron(0 15 * * ? *)`
- `generate`: `cron(0 17 * * ? *)`

All schedules are in UTC.

### How jobs execute

- EventBridge rules run ECS tasks directly.
- Launch type is Fargate.
- The worker ECS service stays at desired count `0`; jobs run from schedules, not a long-running service.
- `import` uses the task default command.
- `update` overrides the command to `update`.
- `generate` overrides the command to `generate`.

## CI deployment

`.github/workflows/deploy.yml` deploys on pushes to `main` and on manual dispatch.

Deployment behavior:

- Pushes two ECR tags: `${GITHUB_SHA}` and `latest`
- Registers a new ECS task definition revision in the `run-job` family
- Pins the ECS container image to the pushed image digest (`@sha256:...`)
- Updates EventBridge rules (`import`, `update`, `generate`) to the new revision

Required GitHub Actions repo variables:

- `AWS_REGION`
- `AWS_REGISTRY_ID`
- `AWS_ROLE_TO_ASSUME`

The assumed role must trust GitHub OIDC (`token.actions.githubusercontent.com`) and allow:

- ECR push/read for `vimcolorschemes/worker`
- ECS task definition read/register (`DescribeTaskDefinition`, `RegisterTaskDefinition`)
- EventBridge target read/update (`ListTargetsByRule`, `PutTargets`)
- `iam:PassRole` for `ecsTaskExecutionRole` and `ecsEventsRole`

## Terraform

Terraform modules and run instructions are in `infra/terraform/README.md`.
