import datetime

from runner import Runner
import printer
import request
import utils


class CleanRunner(Runner):
    def run(self):
        repositories = self.database.get_repositories()
        cleaned_repositories = []
        for repository in repositories:
            printer.info(f"Cleaning {repository['owner']['name']}/{repository['name']}")

            repository = add_archived(repository)

            if "image_urls" not in repository or repository["image_urls"] is None:
                repository["image_urls"] = []

            repository["image_urls"] = utils.remove_duplicates(repository["image_urls"])

            images_dirty, featured_image_dirty = get_dirtiness(repository)

            if images_dirty:
                repository["image_urls"] = []

            if featured_image_dirty:
                repository["featured_image_url"] = None

            if images_dirty or featured_image_dirty:
                repository["cleaned_recently"] = True
                cleaned_repositories.append(
                    f"{repository['owner']['name']}/{repository['name']}"
                )

            self.database.upsert_repository(repository)

        results = {}
        if len(cleaned_repositories) > 0:
            results["cleaned_repositories"] = cleaned_repositories

        return results


def add_archived(repository):
    is_github_url_valid = "github_url" in repository and request.is_url_valid(
        url=repository["github_url"], allow_redirects=False
    )

    initial_archived = False
    if "archived" in repository:
        initial_archived = repository["archived"]

    if not is_github_url_valid and not initial_archived:
        printer.info(
            f"{repository['owner']['name']}/{repository['name']} will be archived"
        )
        repository["archived"] = True
        return repository

    if is_github_url_valid and initial_archived:
        printer.info(
            f"{repository['owner']['name']}/{repository['name']} will be unarchived"
        )
        repository["archived"] = False
        return repository

    return repository


def get_dirtiness(repository):
    images_dirty = False
    featured_image_dirty = False

    for image_url in repository["image_urls"]:
        if not request.is_image_url_valid(image_url):
            images_dirty = True
            break

    if (
        "featured_image_dirty" in repository
        and repository["featured_image_url"] is not None
        and not request.is_image_url_valid(repository["featured_image_url"])
    ):
        featured_image_dirty = True

    return images_dirty, featured_image_dirty
