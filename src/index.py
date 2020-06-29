import datetime
import dateutil.parser as dparser
import os
import time

import db_service
import file_helper
import github
import printer

MAX_IMAGE_COUNT = os.getenv("MAX_IMAGE_COUNT")
MAX_IMAGE_COUNT = int(MAX_IMAGE_COUNT) if MAX_IMAGE_COUNT is not None else 5


if __name__ == "__main__":
    start = time.time()

    printer.break_line()
    printer.break_line()

    last_import_at = db_service.get_last_import_at()

    printer.break_line()
    printer.break_line()

    repositories = github.search_repositories()

    for repository in repositories:
        owner_name = repository["owner"]["name"]
        name = repository["name"]

        db_service.upsert_owner(repository["owner"])
        is_new_repository = db_service.is_repository_new(owner_name, name)

        repository["last_commit_at"] = github.get_last_commit_at(repository)
        last_commit_at = (
            dparser.parse(repository["last_commit_at"], fuzzy=True)
            if repository["last_commit_at"] is not None
            else None
        )

        fetch_images = (
            is_new_repository
            or last_import_at is None
            or last_commit_at > last_import_at
        )

        printer.info(
            "Images will be fetched" if fetch_images else "Image fetching skipped"
        )

        if fetch_images:
            readme_images = file_helper.find_images(
                github.get_readme_file(repository), MAX_IMAGE_COUNT
            )
            repository_images = github.list_repository_images(
                repository, len(readme_images), MAX_IMAGE_COUNT
            )

            repository["image_urls"] = readme_images + repository_images

            printer.info(f"{len(readme_images)} images found in readme")
            printer.info(f"{len(repository_images)} images found in files")

        db_service.upsert_repository(repository)

        printer.break_line()
        printer.break_line()

    end = time.time()

    db_service.create_import(
        {
            "elapsed_time": end - start,
            "import_at": datetime.datetime.now(datetime.timezone.utc),
        }
    )
