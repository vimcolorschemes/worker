<h1 align="center">
  <img alt="vimcolorschemes worker" src="media/logo.png" width="400" />
</h1>
<p align="center" style="border:none">
  I fetch color schemes repositories, and store them. That's about it
</p>

## Getting started

The import script queries Github repositories matching a query, and stores them in a mongoDB database.

This is the data source of [vimcolorschemes](https://github.com/reobin/vimcolorschemes)

### Requirements:

- [python3](https://installpython3.com/)
- [pip](https://pip.pypa.io/en/stable/installing/)
- [mongodb-community](https://docs.mongodb.com/manual/installation/#mongodb-community-edition-installation-tutorials) running at port 27017

### Prepare your environment

#### Create the python virtual environment

Create your virtual environment at the project root:

```shell
python3 -m venv env
```

Then source it:

```shell
source env/bin/activate
```

Install all project dependencies:

```shell
pip install -r requirements.txt
```

#### Get your Github Personnal Access Token

Since GitHub's API has a quite short rate limit for unauthenticated calls (60 for core API calls).
I highly recommend setting up authentication (5000 calls for core API calls) to avoid wait times when you reach the limit.

To do that, you first need to create your personal access token with permissions to read public repositories. Follow instructions on how to do that [here](https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line).

#### Set up the environment variables

A template dotenv file (`.template.env`) is available at root.

Copy it using `cp .template.env .env` and update the values to your needs (read the comments).

> TIP: Read the comments on the template dotenv file.

Then source it:

```shell
source .env
```

### Run the script

To run the script using the default event (`import`), run `python3 src/index.py`.

2 other events are supported: `update` and `clean`. To run them, pass the name of the event as an argument to the python script. Ex.: `python3 src/index.py update`.

To have data ready to use on the app, you should run both `import` and `update` in that order.

[Read more on the events](https://github.com/reobin/vimcolorschemes/wiki/The-Worker)

## Deployment to Lambda

### The environment layer

A Lambda Layer is used to hold all the script dependencies.

When a new dependency is added, or one needs to be updated, the script should be run to build the layer.
The layer then needs to be updloaded to the configured Lambda Layer.

To run the script:

```bash
# at project root
sh create_lambda_layer.sh
```

### The actual script

A Github Action is set up to zip the content of the source directory, and deploy it to AWS Lambda.

All this is done on push to `master`.
