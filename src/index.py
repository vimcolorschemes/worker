import time

import db_service
import github
import worker


def handler(event, context):
    start = time.time()

    last_import_at = db_service.get_last_import_at()

    repositories = github.search_repositories()
    for repository in repositories:
        worker.update_repository(repository, last_import_at)

    end = time.time()

    elapsed_time = end - start

    worker.create_import(elapsed_time)

    worker.call_build_webhook()

    return {"statusCode": 200, "body": f"Elapsed time: {elapsed_time}"}


if __name__ == "__main__":
    handler(None, None)
