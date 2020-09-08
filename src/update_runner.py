import datetime
import functools
import os
import re

import github
import printer
import utils
import request
from runner import Runner

MAX_IMAGE_COUNT = os.getenv("MAX_IMAGE_COUNT")
MAX_IMAGE_COUNT = int(MAX_IMAGE_COUNT) if MAX_IMAGE_COUNT is not None else 5

BUILD_WEBHOOK = os.getenv("BUILD_WEBHOOK")

IMAGE_PATH_REGEX = r"^.*\.(png|jpe?g|webp)$"
POTENTIAL_VIM_COLOR_SCHEME_PATH_REGEX = r"^.*\.(vim|erb)$"

VIM_COLLECTION_THRESHOLD = 15

DAYS_IN_MONTH = 30


class UpdateRunner(Runner):
    def run(self):
        repositories = self.database.get_repositories()

        printer.info(f"Running import on {len(repositories)} repositories")
        printer.break_line()

        github.get_rate_limit()

        for repository in repositories:
            owner_name = repository["owner"]["name"]
            name = repository["name"]

            printer.info(repository["github_url"])

            repository["last_commit_at"] = github.get_last_commit_at(owner_name, name)

            repository["stargazers_count_history"] = update_stargazers_count_history(
                repository
            )

            old_repository = self.database.get_repository(owner_name, name)
            is_fetch_due = compute_is_fetch_due(
                old_repository, repository["last_commit_at"], self.last_job_at
            )
            cleaned_recently = (
                "cleaned_recently" in old_repository
                and old_repository["cleaned_recently"] is True
            )

            if is_fetch_due or (
                "valid" in repository
                and repository["valid"] is True
                and cleaned_recently
            ):
                files = github.get_repository_files(repository)

                if is_fetch_due:
                    printer.info("vim fetch due")
                    repository[
                        "vim_color_scheme_names"
                    ] = get_repository_vim_color_scheme_names(owner_name, name, files)
                    repository["valid"] = len(repository["vim_color_scheme_names"]) > 0
                    repository["vim_fetched_at"] = datetime.datetime.now()
                else:
                    printer.info("vim fetch not due")

                if repository["valid"] is True:
                    printer.info("images fetch is due")
                    old_repository_image_urls = (
                        old_repository["image_urls"]
                        if old_repository is not None and "image_urls" in old_repository
                        else []
                    )
                    repository["image_urls"] = get_repository_image_urls(
                        owner_name, name, files, old_repository_image_urls
                    )
                    repository["images_fetched_at"] = datetime.datetime.now()
                    repository["cleaned_recently"] = False
                else:
                    printer.info("images fetch not is due")

            if "valid" in repository and repository["valid"] is True:
                printer.info("Repository is a valid vim color scheme")
            else:
                printer.info("Repository is not a valid vim color scheme")
                repository["image_urls"] = []

            self.database.upsert_repository(repository)

        call_build_webhook()

        return {"repository_count": len(repositories)}


def call_build_webhook():
    if BUILD_WEBHOOK is not None and BUILD_WEBHOOK != "":
        printer.break_line()
        printer.info("Starting website build")
        response = request.post(BUILD_WEBHOOK, is_json=False,)
        printer.info(f"Response: {response}")
        printer.break_line()


def compute_is_fetch_due(old_repository, last_commit_at, last_job_at):
    # if the repository is new or has never been fetched before, fetch
    if old_repository is None or "vim_fetched_at" not in old_repository:
        return True

    # if the repository was deemed invalid before, don't fetch
    if "valid" in old_repository and old_repository["valid"] == False:
        return False

    # if no prior update, fetch
    if last_job_at is None:
        return True

    # fetch if the repository was updated after last update
    return last_commit_at > last_job_at


def get_repository_image_urls(owner_name, name, files, old_image_urls):
    image_urls = old_image_urls
    max_image_count_left = MAX_IMAGE_COUNT - len(image_urls)

    # search readme
    if max_image_count_left > 0:
        image_urls.extend(
            get_image_urls_from_readme(
                owner_name, name, old_image_urls, max_image_count_left
            )
        )
        max_image_count_left -= len(image_urls)

    # search file tree
    if max_image_count_left > 0:
        image_files = list(
            filter(lambda file: re.match(IMAGE_PATH_REGEX, file["path"]), files)
        )
        file_tree_image_urls = get_new_image_urls(
            list(
                map(
                    lambda file: utils.build_raw_blog_github_url(
                        owner_name, name, file["path"]
                    ),
                    image_files,
                )
            ),
            image_urls,
        )[0:max_image_count_left]
        image_urls.extend(file_tree_image_urls)

    return utils.remove_duplicates(image_urls)


def get_new_image_urls(potentially_new_image_urls, old_image_urls):
    return [
        image_url
        for image_url in potentially_new_image_urls
        if image_url not in old_image_urls
    ]


def get_image_urls_from_readme(owner_name, name, old_image_urls, max_image_count_left):
    readme_file = github.get_readme_file(owner_name, name)
    image_urls = utils.find_image_urls(
        readme_file, old_image_urls, max_image_count_left
    )
    return image_urls


def update_stargazers_count_history(repository):
    history = (
        repository["stargazers_count_history"]
        if "stargazers_count_history" in repository
        else []
    )
    if history is None:
        history = []

    if len(history) > 0:
        history.sort(key=lambda entry: entry["date"])

    today = datetime.date.today().isoformat()

    matching_indexes = [
        index if entry["date"] == today else -1 for index, entry in enumerate(history)
    ]
    matching_indexes.sort(reverse=True)
    for index in list(filter(lambda index: index != -1, matching_indexes)):
        del history[index]

    if len(history) >= DAYS_IN_MONTH:
        del history[0]

    history.append(
        {"date": today, "stargazers_count": repository["stargazers_count"],}
    )

    return history


def get_repository_vim_color_scheme_names(owner_name, name, files):
    vim_files = list(
        filter(
            lambda file: re.match(POTENTIAL_VIM_COLOR_SCHEME_PATH_REGEX, file["path"]),
            files,
        )
    )

    vim_color_scheme_names = []
    for vim_file in vim_files:
        vim_color_scheme_name = get_vim_color_scheme_name(owner_name, name, vim_file)
        if (
            vim_color_scheme_name is not None
            and vim_color_scheme_name != ""
            and vim_color_scheme_name not in vim_color_scheme_names
        ):
            vim_color_scheme_names.append(vim_color_scheme_name)
            if len(vim_color_scheme_names) > VIM_COLLECTION_THRESHOLD:
                break

    if len(vim_color_scheme_names) <= VIM_COLLECTION_THRESHOLD:
        return vim_color_scheme_names

    printer.info(
        "Repository contains too many vim color schemes; probably a collection"
    )
    return []


def get_vim_color_scheme_name(owner_name, name, file):
    vim_color_scheme_name = None
    response = request.get(
        utils.build_raw_blog_github_url(owner_name, name, file["path"]), is_json=False,
    )
    file_content = response.text if response is not None else ""

    match = re.search(
        r"let (g:)?colors?_name ?= ?('|\")([a-zA-Z-_0-9]+)('|\")", file_content
    )
    if match is not None:
        vim_color_scheme_name = match.group(3)
        printer.info(f"{name} vim color scheme name is {vim_color_scheme_name}")

    return vim_color_scheme_name
