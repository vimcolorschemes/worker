<h1 align="center">
  <img alt="vimcolorschemes worker" src="https://github.com/vimcolorschemes/worker/blob/media/logo.png?raw=true" width="400" />
</h1>
<p align="center" style="border:none">
  I search for vim color schemes, and store them. That's about it.
</p>

## Getting started

The worker is a CLI used to import and manage the data for [vimcolorschemes](https://github.com/reobin/vimcolorschemes).

### Requirements:

- [mongodb-community](https://docs.mongodb.com/manual/installation/#mongodb-community-edition-installation-tutorials)

_Note_: The MongoDB database can also be ran from [the app docker setup](https://docs.vimcolorschemes.com/#/installation-guide?id=_1-docker).

### Set up the environment variables

A template dotenv file (`.env.example`) is available at root.

Copy it using `cp .env.example .env` and update the values to your needs.

> TIP: Read the comments on the template dotenv file.

The `.env` is automatically picked up by CLI when it runs.

#### GitHub queries

Since GitHub's API has a quite short rate limit for unauthenticated calls (60 for core API calls).
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

#### update

Fetch the necessary data for the repositories

```shell
bin/start update
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
