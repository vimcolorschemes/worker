import github
from runner import Runner


class ImportRunner(Runner):
    # Search for repositories, store
    def run(self):
        repositories = github.search_repositories()

        for repository in repositories:
            self.database.upsert_repository(repository)

        return {"repository_count": len(repositories)}
