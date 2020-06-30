# vimcolorschemes worker

Python cron job updating data for vimcolorschemes

## Getting started

The import script queries Github repositories matching some criterias, and stores them in a mongoDB database.

The database is then queried by the Gatsby App to build the website.

### Requirements:

- python3
  `curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py`

When your machine is ready, you need to create your virtual environment at the project root:

```shell
python3 -m venv env
```

Then source your virtual env:

```shell
source env/bin/activate
```

Install all project dependencies:

```shell
pip install -r requirements.txt
```

You should be good to go to run the script, but there's a couple things left to talk about.

Github API has a quite short rate limit for unauthenticated calls (60 for core API calls).
I highly recommend setting up authentication (5000 calls for core API calls) to avoid wait times when you reach the limit.

To do that, you first need to create your personal access token with permissions to read public repositories. Follow instructions on how to do that [here](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line).

The last thing you need to do is setup your environment variables.

Copy the template environment file using `cp .template.env .env`, and then update the values to your needs

> Note: The non-Github stuff is a set of recommended values to make development effective. They can be altered with.

Then source it:

```shell
source .env
```

Start the script with `python3 src/index.py`

## Deployment to Lambda

### Environment

A Lambda Layer is used to hold all the script dependencies.

When a new dependency is added, or one needs to be updating, the script needs to be ran to build the layer.
The layer then needs to be updloaded to the configured Lambda Layer.

To run the script:

```bash
# at project root
sh create_lambda_layer.sh
```

### Script

A Github Action is set up to zip the content of the source directory, and deploy it to AWS Lambda.

All this is done on push to `master`.
