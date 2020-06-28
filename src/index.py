from datetime import datetime
import dateutil.parser as dparser
import os
import time

import api
import file_helper
import github
import printer

MAX_IMAGE_COUNT = os.getenv("MAX_IMAGE_COUNT")
MAX_IMAGE_COUNT = int(MAX_IMAGE_COUNT) if MAX_IMAGE_COUNT is not None else 5


if __name__ == "__main__":
    start = time.time()

    printer.break_line()
    printer.break_line()

    last_import_at = api.get_last_import_at()

    printer.break_line()
    printer.break_line()

    repositories = github.search_repositories()

    for repository in repositories:
        existing_repository = api.get_repository_by_github_id(repository["github_id"])

        repository["last_commit_at"] = github.get_last_commit_at(repository)
        last_commit_at = dparser.parse(repository["last_commit_at"], fuzzy=True)

        fetch_images = (
            last_import_at is None
            or last_commit_at > last_import_at
            or existing_repository is None
        )

        printer.info(
            "Images will be fetched" if fetch_images else "Image fetching skipped"
        )

        new_repository = None
        owner_name = repository["owner"]["name"]
        owner = api.get_owner_by_name(owner_name)
        if owner is None:
            printer.info(f"owner {owner_name} does not exist")
            owner = api.create_owner({"name": owner_name})
        repository = {**repository, "owner": owner["id"]}

        if existing_repository is None:
            new_repository = api.create_repository(repository)
        else:
            if fetch_images:
                api.delete_images(existing_repository["images"])
            new_repository = api.update_repository(
                existing_repository["id"], repository
            )

        if fetch_images:
            readme_images = file_helper.find_images(
                github.get_readme_file(repository), MAX_IMAGE_COUNT
            )
            repository_images = github.list_repository_images(
                repository, len(readme_images), MAX_IMAGE_COUNT
            )
            api.upload_repository_images(
                new_repository, readme_images + repository_images
            )
            printer.info(f"{len(readme_images)} images found in readme")
            printer.info(f"{len(repository_images)} images found in files")

        printer.break_line()
        printer.break_line()

    end = time.time()

    api.create_import({"elapsed_time": end - start})
