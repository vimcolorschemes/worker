from runner import Runner
import printer
import request


class CleanRunner(Runner):
    def run(self):
        printer.info("Cleaning")
        repositories = self.database.get_repositories()
        for repository in repositories:
            printer.info(f"Cleaning {repository['owner']['name']}/{repository['name']}")

            image_urls = repository["image_urls"] if "image_urls" in repository else []
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

            if image_removed_count > 0:
                self.database.upsert_repository(repository)
            else:
                printer.info("Repository not updated")
                printer.break_line()
