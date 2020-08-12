import base64
import datetime
import math
import os
import pytz
import re
import sys
import time

from requests.auth import HTTPBasicAuth

import request
import printer
import utils

GITHUB_USERNAME = os.getenv("GITHUB_USERNAME")
GITHUB_TOKEN = os.getenv("GITHUB_TOKEN")

ITEMS_PER_PAGE = 100
BASE_URL = "https://api.github.com"

GITHUB_BASIC_AUTH = (
    HTTPBasicAuth(GITHUB_USERNAME, GITHUB_TOKEN)
    if GITHUB_USERNAME and GITHUB_TOKEN
    else None
)

GITHUB_API_HARD_LIMIT = 1000
REPOSITORY_LIMIT = os.getenv("REPOSITORY_LIMIT")
REPOSITORY_LIMIT = int(REPOSITORY_LIMIT) if REPOSITORY_LIMIT is not None else None
REPOSITORY_LIMIT = min(GITHUB_API_HARD_LIMIT, REPOSITORY_LIMIT)

this = sys.modules[__name__]
this.remaining_github_api_calls = None
this.github_api_rate_limit_reset = None
this.file_tree_requests_left = 20


def convert_github_string_datetime(d):
    # GitHub format: 2020-07-16T19:23:44Z
    unaware = datetime.datetime.strptime(d, "%Y-%m-%dT%H:%M:%SZ")
    aware = unaware.replace(tzinfo=pytz.UTC)
    return aware


def get_rate_limit():
    printer.info("GET GitHub API rate limit")

    data = request.get(f"{BASE_URL}/rate_limit", auth=GITHUB_BASIC_AUTH)

    this.remaining_github_api_calls = data["resources"]["core"]["remaining"]
    this.github_api_rate_limit_reset = data["resources"]["core"]["reset"]

    printer.info(f"{this.remaining_github_api_calls} remaining calls for GitHub API")
    printer.break_line()
    printer.break_line()


def sleep_until_reset():
    while this.remaining_github_api_calls <= 1:
        now = int(time.time())
        time_until_reset = max(0, this.github_api_rate_limit_reset - now)
        safety_buffer = 100
        sleep_time = time_until_reset + safety_buffer

        printer.warning(
            f"Github API's rate limit reached. Need to sleep for {sleep_time} seconds"
        )

        printer.start_sleeping(sleep_time)
        get_rate_limit()


# This calls the basic request helper's function get, but also handles the Github API's rate limit check
def github_core_get(url, params=None, log=None):
    if (
        this.remaining_github_api_calls is None
        or this.github_api_rate_limit_reset is None
    ):
        get_rate_limit()

    if this.remaining_github_api_calls <= 1:
        sleep_until_reset()

    if log is not None:
        printer.info(log)

    data = request.get(url=url, params=params, auth=GITHUB_BASIC_AUTH)

    return data


def list_repositories_of_page(query, page=1):
    items_per_page = min(REPOSITORY_LIMIT, ITEMS_PER_PAGE)
    search_path = "search/repositories"
    base_search_params = {"per_page": items_per_page}

    data = github_core_get(
        url=f"{BASE_URL}/{search_path}",
        params={"q": query, "page": page, **base_search_params},
        log=f"GET repositories (page: {page})",
    )

    repositories = list(map(map_response_item_to_repository, data["items"]))
    total_count = data["total_count"]

    return repositories, total_count


# Fetches github repositories with defined queries.
# If more than 100 repositories, it will search all pages one by one.
def search_repositories():
    queries = [
        "vim color scheme",
        "vim colorscheme",
        "vim colour scheme",
        "vim colourscheme",
    ]

    repositories = []

    for query in queries:
        query = f"{query} NOT dotfiles sort:stars stars:>1"

        first_page_repositories, total_count = list_repositories_of_page(query)
        repositories.extend(first_page_repositories)

        fetched_repository_count = (
            min(REPOSITORY_LIMIT, total_count)
            if REPOSITORY_LIMIT is not None
            else total_count
        )

        printer.info(f"{fetched_repository_count} fetched for query {query}")

        page_count = math.ceil(fetched_repository_count / ITEMS_PER_PAGE)

        for page in range(2, page_count + 1):
            current_page_repositories, _total_count = list_repositories_of_page(
                query, page
            )
            repositories.extend(current_page_repositories)

    printer.info(f"{len(repositories)} repositories will be processed")
    printer.break_line()
    printer.break_line()

    return repositories


# Keeps only the stuff we need for the app
def map_response_item_to_repository(response_item):
    return {
        "github_id": response_item["id"],
        "name": response_item["name"],
        "description": response_item["description"],
        "default_branch": response_item["default_branch"],
        "github_url": response_item["html_url"],
        "homepage_url": response_item["homepage"],
        "stargazers_count": response_item["stargazers_count"],
        "pushed_at": convert_github_string_datetime(response_item["pushed_at"]),
        "github_created_at": convert_github_string_datetime(
            response_item["created_at"]
        ),
        "owner": {
            "name": response_item["owner"]["login"],
            "avatar_url": response_item["owner"]["avatar_url"],
        },
    }


def get_last_commit_at(owner_name, name):
    commits_path = f"repos/{owner_name}/{name}/commits"

    commits = github_core_get(
        f"{BASE_URL}/{commits_path}", log=f"GET {owner_name}/{name} last commit at"
    )

    if not commits or len(commits) == 0:
        return None

    last_commit_data = commits[0]

    if (
        "commit" in last_commit_data
        and "committer" in last_commit_data["commit"]
        and "date" in last_commit_data["commit"]["committer"]
    ):
        string_datetime = last_commit_data["commit"]["committer"]["date"]
        return convert_github_string_datetime(string_datetime)

    return None


def list_objects_of_tree(owner_name, name, tree_sha):
    tree_path = f"repos/{owner_name}/{name}/git/trees/{tree_sha}"
    data = github_core_get(
        f"{BASE_URL}/{tree_path}",
        log=f"GET {owner_name}/{name} objects of tree {tree_path}",
    )
    return data["tree"]


def get_tree_path(tree_object):
    path = tree_object["path"]
    current_tree = tree_object
    while "parent_tree_object" in current_tree:
        current_tree = current_tree["parent_tree_object"]
        path = f"{current_tree['path']}/{path}"
    return path


def get_files_of_tree(owner_name, name, tree_sha, tree_path):
    if this.file_tree_requests_left <= 0:
        return []

    tree_objects = list_objects_of_tree(owner_name, name, tree_sha)
    this.file_tree_requests_left -= 1

    files = [obj for obj in tree_objects if obj["type"] == "blob"]
    trees = [obj for obj in tree_objects if obj["type"] == "tree"]

    tree_files = []
    for tree in trees:
        tree_files = tree_files + get_files_of_tree(
            owner_name, name, tree["sha"], f"{tree_path}/{tree['path']}"
        )

    return [
        {**file, "path": f"{tree_path}/{file['path']}"} for file in files
    ] + tree_files


def get_repository_files(repository):
    owner_name = repository["owner"]["name"]
    name = repository["name"]

    printer.info(f"Getting files for {owner_name}/{name}")

    this.file_tree_requests_left = 20

    return get_files_of_tree(
        owner_name, name, repository["default_branch"], repository["default_branch"]
    )


def get_readme_file(owner_name, name):
    if owner_name is None or name is None:
        return ""

    get_readme_path = f"repos/{owner_name}/{name}/readme"

    readme_data = github_core_get(
        f"{BASE_URL}/{get_readme_path}", log=f"GET {owner_name}/{name} readme"
    )

    if readme_data is None:
        return ""

    return utils.decode_base64(readme_data["content"])
