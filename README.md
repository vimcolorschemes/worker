<h1 align="center">
  <img alt="vimcolorschemes worker" src="https://github.com/vimcolorschemes/worker/blob/media/logo.png?raw=true" width="400" />
</h1>
<p align="center" style="border:none">
  I search for vim color schemes, and store them. That's about it.
</p>

## Getting started

The worker is a CLI used to import and manage the data for [vimcolorschemes](https://github.com/vimcolorschemes/vimcolorschemes).

### Requirements:

- [mongodb-community](https://docs.mongodb.com/manual/installation/#mongodb-community-edition-installation-tutorials)
- [golang](https://go.dev)

_Note_: The MongoDB database can also be ran from [the app docker setup](https://docs.vimcolorschemes.com/#/installation-guide?id=_1-docker).

### Set up the environment variables

Update the values in `.env` to your needs.

> TIP: Read the comments on the dotenv file.

The `.env` is automatically picked up by CLI when it runs.

#### Github queries

Since Github's API has a quite short rate limit for unauthenticated calls (60 for core API calls).
I highly recommend setting up authentication (5000 calls for core API calls) to avoid wait times when you reach the limit.

To do that, you first need to create your personal access token with permissions to read public repositories. Follow instructions on how to do that [here](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line).

### Run a job

To run a job, use the `bin/start` script:

```shell
bin/start <job>
```

[Read about on the jobs](https://docs.vimcolorschemes.com/#/the-worker)

#### import

Import repositories into the database

```shell
bin/start import
```

Import only a specific repository using the `--repo` option.

```shell
bin/start import --repo morhetz/gruvbox
```

#### update

Fetch the necessary data for the repositories

```shell
bin/start update
```

Force a full update of all the repositories by using the `--force` option.

```shell
bin/start update --force
```

Update only a specific repository using the `--repo` option.

```shell
bin/start update --repo morhetz/gruvbox
```

#### generate

Generate color data for the vim color scheme previews

```shell
bin/start generate
```

Force a full generation of all the repositories by using the `--force` option.

```shell
bin/start generate --force
```

Generate preview data for only a specific repository using the `--repo` option.

```shell
bin/start generate --repo morhetz/gruvbox
```

### Run tests

```shell
bin/test
```

`go test` flags can be used:

```shell
bin/test --cover
```

### Lint

[golint](https://github.com/golang/lint) is configured to run using `bin/lint`.

### Format code

`go fmt` can be easily used on all code using `bin/format`.

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

### Deploying updates

Publish a new worker image to ECR; subsequent scheduled tasks run with the updated image.

### CI deployment

This repo includes `.github/workflows/deploy.yml` to deploy on every push to `main` (and on manual dispatch).

Deployment behavior:

- Pushes two ECR tags: `${GITHUB_SHA}` and `latest`
- Registers a new ECS task definition revision in the `run-job` family
- Pins the ECS container image to the pushed image digest (`@sha256:...`)
- Updates EventBridge rules (`import`, `update`, `generate`) to the new revision

Set these repository-level GitHub Actions variables:

- `AWS_REGION`
- `AWS_REGISTRY_ID`
- `AWS_ROLE_TO_ASSUME`

The assumed role must trust GitHub OIDC (`token.actions.githubusercontent.com`) and allow:

- ECR push/read for `vimcolorschemes/worker`
- ECS task definition read/register (`DescribeTaskDefinition`, `RegisterTaskDefinition`)
- EventBridge target read/update (`ListTargetsByRule`, `PutTargets`)
- `iam:PassRole` for task roles used by `run-job`
