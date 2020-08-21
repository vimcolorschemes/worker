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

VIM_COLLECTION_THRESHOLD = 20


class UpdateRunner(Runner):
    def run(self):
        repositories = self.database.get_repositories()

        printer.info(f"Running import on {len(repositories)} repositories")
        printer.break_line()

        for repository in repositories:
            owner_name = repository["owner"]["name"]
            name = repository["name"]

            printer.info(repository["github_url"])

            repository["last_commit_at"] = github.get_last_commit_at(owner_name, name)

            old_repository = self.database.get_repository(owner_name, name)
            if is_fetch_due(
                old_repository, repository["last_commit_at"], self.last_job_at
            ):
                printer.info("Fetch is due")
                files = github.get_repository_files(repository)
                repository[
                    "vim_color_scheme_file_paths"
                ] = get_repository_vim_color_scheme_file_paths(owner_name, name, files)
                repository["valid"] = len(repository["vim_color_scheme_file_paths"]) > 0
                if repository["valid"]:
                    old_repository_image_urls = (
                        old_repository["image_urls"]
                        if old_repository is not None and "image_urls" in old_repository
                        else []
                    )
                    repository["image_urls"] = get_repository_image_urls(
                        owner_name, name, files, old_repository_image_urls
                    )
                else:
                    repository["image_urls"] = []
                    printer.info("Repository is not a valid vim color scheme")
            else:
                printer.info("Repository is not due for a content fetch")

            self.database.upsert_repository(repository)

        call_build_webhook()


def call_build_webhook():
    if BUILD_WEBHOOK is not None and BUILD_WEBHOOK != "":
        printer.break_line()
        printer.info("Starting website build")
        response = request.post(BUILD_WEBHOOK, is_json=False,)
        printer.info(f"Response: {response}")
        printer.break_line()


def is_fetch_due(old_repository, last_commit_at, last_job_at):
    # if the repository is new, fetch
    if old_repository is None:
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
    max_image_count_left = MAX_IMAGE_COUNT - len(old_image_urls)

    if max_image_count_left <= 0:
        return old_image_urls

    readme_file = github.get_readme_file(owner_name, name)
    image_urls = utils.find_image_urls(readme_file, max_image_count_left)

    max_image_count_left -= len(image_urls)

    if max_image_count_left <= 0:
        return image_urls

    image_files = list(
        filter(lambda file: re.match(IMAGE_PATH_REGEX, file["path"]), files)
    )[0:max_image_count_left]

    image_urls = image_urls + list(
        map(
            lambda file: utils.build_raw_blog_github_url(
                owner_name, name, file["path"]
            ),
            image_files,
        )
    )
    return image_urls


def get_repository_vim_color_scheme_file_paths(owner_name, name, files):
    vim_files = list(
        filter(
            lambda file: re.match(POTENTIAL_VIM_COLOR_SCHEME_PATH_REGEX, file["path"]),
            files,
        )
    )

    if len(vim_files) < VIM_COLLECTION_THRESHOLD:
        vim_color_scheme_files = list(
            filter(
                lambda file: utils.is_vim_color_scheme(owner_name, name, file),
                vim_files,
            )
        )
        return list(map(lambda file: file["path"], vim_color_scheme_files))

    return []
