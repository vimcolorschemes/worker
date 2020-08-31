import datetime

from runner import Runner
import printer
import request
import utils


class CleanRunner(Runner):
    def run(self):
        repositories = self.database.get_repositories()
        total_image_removed_count = 0
        affected_repositories = []
        for repository in repositories:
            printer.info(f"Cleaning {repository['owner']['name']}/{repository['name']}")

            clean_image_urls, image_removed_count = get_clean_image_urls(repository)
            (
                clean_featured_image_url,
                is_featured_image_removed,
            ) = get_clean_featured_image_url(repository, clean_image_urls)

            repository["image_urls"] = clean_image_urls
            repository["featured_image_url"] = clean_featured_image_url

            if image_removed_count > 0:
                repository["cleaned_recently"] = True
                affected_repositories.append(
                    f"{repository['owner']['name']}/{repository['name']}"
                )

            total_image_removed_count += image_removed_count + (
                1 if is_featured_image_removed else 0
            )

            self.database.upsert_repository(repository)

        results = {
            "image_removed_count": total_image_removed_count,
        }
        if len(affected_repositories) > 0:
            results["affected_repositories"] = affected_repositories

        return results


def get_clean_image_urls(repository):
    printer.info("Cleaning image urls")

    image_urls = repository["image_urls"] if "image_urls" in repository else []
    if image_urls is None:
        image_urls = []

    initial_count = len(image_urls)
    printer.info(f"Initial image count: {initial_count}")

    # remove duplicates
    printer.info("Removing duplicates")
    image_urls = utils.remove_duplicates(image_urls)

    # remove no-longer valid urls
    printer.info("Removing invalid urls")
    image_urls = list(
        filter(lambda image_url: request.is_image_url_valid(image_url), image_urls)
    )

    image_removed_count = initial_count - len(image_urls)
    printer.info(f"Removed {image_removed_count} images")

    return image_urls, image_removed_count


def get_clean_featured_image_url(repository, clean_image_urls):
    printer.info("Cleaning featured image url")
    initial_featured_image_url = (
        repository["featured_image_url"] if "featured_image_url" in repository else None
    )
    featured_image_url = initial_featured_image_url
    if featured_image_url is not None and featured_image_url not in clean_image_urls:
        featured_image_url = None
    return featured_image_url, initial_featured_image_url != featured_image_url
