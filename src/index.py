import time
import os

from database import Database
from worker import Worker
import github

COLOR_SCHEME_QUERY = os.getenv("COLOR_SCHEME_QUERY")

DATABASE_HOST = os.getenv("DATABASE_HOST")
DATABASE_USERNAME = os.getenv("DATABASE_USERNAME")
DATABASE_PASSWORD = os.getenv("DATABASE_PASSWORD")
if DATABASE_PASSWORD == "":
    DATABASE_PASSWORD = None

connection = {"host": DATABASE_HOST}
if DATABASE_USERNAME is not None and DATABASE_USERNAME != "":
    connection["username"] = DATABASE_USERNAME
if DATABASE_PASSWORD is not None and DATABASE_PASSWORD != "":
    connection["password"] = DATABASE_PASSWORD


def handler(event, context):
    start = time.time()

    if "host" in event:
        connection["host"] = event["host"]
    if "username" in event:
        connection["username"] = event["username"]
    if "password" in event:
        connection["password"] = event["password"]

    database_instance = Database(**connection)
    worker_instance = Worker(database_instance)

    last_import_at = worker_instance.get_last_import_at()

    query = event["query"] if "query" in event else COLOR_SCHEME_QUERY
    repositories = github.search_repositories(query)
    for repository in repositories:
        worker_instance.update_repository(repository, last_import_at)

    end = time.time()

    elapsed_time = end - start

    worker_instance.create_import(elapsed_time)

    worker_instance.call_build_webhook()

    return {"statusCode": 200, "body": f"Elapsed time: {elapsed_time}"}

if __name__ == "__main__":
    handler({}, None)
