import datetime
import os
import re

import github
import printer
import request
import utils

MAX_IMAGE_COUNT = os.getenv("MAX_IMAGE_COUNT")
MAX_IMAGE_COUNT = int(MAX_IMAGE_COUNT) if MAX_IMAGE_COUNT is not None else 5

BUILD_WEBHOOK = os.getenv("BUILD_WEBHOOK")

IMAGE_PATH_REGEX = r"^.*\.(png|jpe?g|webp)$"
POTENTIAL_VIM_COLOR_SCHEME_PATH_REGEX = r"^.*\.(vim|erb)$"

VIM_COLLECTION_THRESHOLD = 20


class Worker:
    def __init__(self, database):
        self.database = database
        self.last_import_at = self.database.get_last_import_at()

    # Search for repositories, validate, store
    def run_import(self):
        repositories = github.search_repositories()

        for repository in repositories:
            owner_name = repository["owner"]["name"]
            name = repository["name"]

            printer.info(repository["github_url"])

            repository["last_commit_at"] = github.get_last_commit_at(owner_name, name)

            old_repository = self.database.get_repository(owner_name, name)
            if self.is_update_due(old_repository, repository["last_commit_at"]) or True:
                printer.info("Update is due")
                files = github.get_repository_files(repository)
                repository[
                    "vim_color_scheme_file_paths"
                ] = self.get_vim_color_scheme_file_paths(owner_name, name, files)
                repository["valid"] = len(repository["vim_color_scheme_file_paths"]) > 0
                if repository["valid"]:
                    repository["image_urls"] = self.get_image_urls(owner_name, name, files)

            self.database.upsert_repository(repository)

    def get_image_urls(self, owner_name, name, files):
        readme_file = github.get_readme_file(owner_name, name)
        image_urls = utils.find_image_urls(readme_file)
        image_files = list(
            filter(lambda file: re.match(IMAGE_PATH_REGEX, file["path"]), files)
        )
        image_urls = image_urls + list(
            map(
                lambda file: f"https://raw.githubusercontent.com/{owner_name}/{name}/{file['path']}",
                image_files,
            )
        )
        return image_urls

    def get_vim_color_scheme_file_paths(self, owner_name, name, files):
        vim_files = list(
            filter(
                lambda file: re.match(
                    POTENTIAL_VIM_COLOR_SCHEME_PATH_REGEX, file["path"]
                ),
                files,
            )
        )

        if len(vim_files) < VIM_COLLECTION_THRESHOLD:
            vim_color_scheme_files = list(
                filter(
                    lambda file: self.is_vim_color_scheme(owner_name, name, file),
                    vim_files,
                )
            )
            return list(map(lambda file: file["path"], vim_color_scheme_files))

        return []

    def is_vim_color_scheme(self, owner_name, name, file):
        url = f"https://raw.githubusercontent.com/{owner_name}/{name}/{file['path']}"
        response = request.get(url, is_json=False)
        file_content = response.text if response is not None else ""
        return "colors_name" in file_content

    def is_update_due(self, old_repository, last_commit_at):
        # if the repository is new, update
        if old_repository is None:
            return True

        # if the repository was deemed invalid before, don't update
        if "valid" in old_repository and old_repository["valid"] == False:
            return False

        # update if the repository was updated after last import
        return last_commit_at > self.last_import_at

    # Clean out image URLs that are not valid anymore
    def run_clean(self):
        repositories = []
        for repository in repositories:
            print("")
            # validate image URLs

    # Salvage or update repositories not handled anymore by the import
    def run_salvage(self):
        repositories = []
        for repository in repositories:
            print("")
            # valid, image_urls = search file tree
            # upsert
