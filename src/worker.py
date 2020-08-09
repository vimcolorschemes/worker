import datetime
import os

import github
import printer
import request
import utils

MAX_IMAGE_COUNT = os.getenv("MAX_IMAGE_COUNT")
MAX_IMAGE_COUNT = int(MAX_IMAGE_COUNT) if MAX_IMAGE_COUNT is not None else 5

BUILD_WEBHOOK = os.getenv("BUILD_WEBHOOK")


class Worker:
    def __init__(self, database_instance):
        self.database_instance = database_instance

    def update_repository(self, repository, last_import_at):
        owner_name = repository["owner"]["name"]
        name = repository["name"]

        is_repository_new, old_repository = self.database_instance.is_repository_new(
            owner_name, name
        )

        repository["last_commit_at"] = github.get_last_commit_at(repository)

        if self.should_fetch_images(
            repository["last_commit_at"], last_import_at, is_repository_new
        ):
            repository = self.fetch_images(repository, old_repository)

        self.database_instance.upsert_repository(repository)

    def should_fetch_images(self, last_commit_at, last_import_at, is_repository_new):
        images_will_be_fetched = (
            is_repository_new
            or last_import_at is None
            or last_commit_at > last_import_at
        )
        printer.info(
            "Images will be fetched"
            if images_will_be_fetched
            else "Image fetching skipped"
        )
        return images_will_be_fetched

    def fetch_images(self, repository, old_repository):
        image_urls = (
            list(
                filter(
                    lambda url: request.is_image_url_valid(url),
                    old_repository["image_urls"],
                )
            )
            if old_repository is not None
            else []
        )

        image_count_to_find = MAX_IMAGE_COUNT - len(image_urls)

        readme_image_urls = (
            utils.find_image_urls(
                github.get_readme_file(repository), image_count_to_find, image_urls
            )
            if image_count_to_find > 0
            else []
        )

        image_urls = image_urls + readme_image_urls
        image_count_to_find = MAX_IMAGE_COUNT - len(image_urls)

        repository_image_urls = (
            github.list_repository_image_urls(
                repository, image_count_to_find, image_urls
            )
            if image_count_to_find > 0
            else []
        )

        image_urls = image_urls + repository_image_urls

        repository["image_urls"] = list(map(utils.urlify, image_urls))

        printer.info(f"{len(readme_image_urls)} images found in readme")
        printer.info(f"{len(repository_image_urls)} images found in files")

        return repository

    def create_import(self, elapsed_time):
        import_data = {
            "elapsed_time": elapsed_time,
            "import_at": datetime.datetime.now(datetime.timezone.utc),
        }

        self.database_instance.create_import(import_data)

    def call_build_webhook(self):
        if BUILD_WEBHOOK is not None and BUILD_WEBHOOK != "":
            printer.break_line()
            printer.info("Starting website build")
            response = request.post(BUILD_WEBHOOK, is_json=False,)
            printer.info(f"Response: {response}")
            printer.break_line()

    def get_last_import_at(self):
        return self.database_instance.get_last_import_at()

    def clean_repositories(self):
        repositories = self.database_instance.get_repositories()
        total_image_removed_count = 0
        for repository in repositories:
            printer.info(f"Cleaning {repository['owner']['name']}/{repository['name']}")

            image_urls = repository["image_urls"]
            if image_urls is None:
                image_urls = []
            initial_count = len(image_urls)
            printer.info(f"Initial image count: {initial_count}")

            # remove duplicates
            printer.info("Removing duplicates")
            image_urls = list(dict.fromkeys(image_urls))

            # remove no-longer valid urls
            printer.info("Removing invalid urls")
            image_urls = list(
                filter(
                    lambda image_url: request.is_image_url_valid(image_url), image_urls
                )
            )

            final_count = len(image_urls)
            printer.info(f"Final image count: {final_count}")

            featured_image_url = (
                repository["featured_image_url"]
                if "featured_image_url" in repository
                else None
            )
            if featured_image_url not in image_urls:
                featured_image_url = None

            repository["image_urls"] = image_urls
            repository["featured_image_url"] = featured_image_url

            image_removed_count = final_count - initial_count
            total_image_removed_count += image_removed_count

            if image_removed_count > 0:
                self.database_instance.upsert_repository(repository)
            else:
                printer.info("Repository not updated")
                printer.break_line()

        return total_image_removed_count
