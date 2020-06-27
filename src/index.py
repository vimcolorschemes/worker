from datetime import datetime
import dateutil.parser as dparser
import os
import time

import api
import github_helper
import file_helper

MAX_IMAGE_COUNT = os.getenv("MAX_IMAGE_COUNT")
MAX_IMAGE_COUNT = int(MAX_IMAGE_COUNT) if MAX_IMAGE_COUNT is not None else 5


if __name__ == "__main__":
    start = time.time()

    last_import_at = api.get_last_import_at()

    repositories = github_helper.search_repositories()

    for repository in repositories:
        existing_repository = api.get_repository_by_github_id(repository["github_id"])

        repository["last_commit_at"] = github_helper.get_last_commit_at(repository)
        last_commit_at = dparser.parse(repository["last_commit_at"], fuzzy=True)

        refetch_images = (
            not last_import_at
            or last_commit_at > last_import_at
            or existing_repository is None
        )

        new_repository = None
        if existing_repository is None:
            owner_name = repository["owner"]["name"]
            owner = api.get_owner_by_name(owner_name)
            if owner is None:
                owner = api.create_owner({"name": owner_name})
            new_repository = api.create_repository({**repository, "owner": owner["id"]})
        else:
            if refetch_images:
                api.delete_images(existing_repository["images"])
            new_repository = api.update_repository(
                existing_repository["id"], repository
            )

        if refetch_images:
            readme_images = file_helper.find_images(
                github_helper.get_readme_file(repository), MAX_IMAGE_COUNT
            )
            repository_images = github_helper.list_repository_images(
                repository, len(readme_images), MAX_IMAGE_COUNT
            )
            api.upload_repository_images(
                new_repository, readme_images + repository_images
            )

    end = time.time()

    api.create_import({"elapsed_time": end - start})
