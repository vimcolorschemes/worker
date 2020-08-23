import os
import sys
import time
from enum import Enum

from database import Database
import github
import printer

from import_runner import ImportRunner
from clean_runner import CleanRunner
from update_runner import UpdateRunner


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


def start(job):
    start = time.time()

    database_instance = Database(**connection)

    if job == "clean":
        runner = CleanRunner(database_instance, "clean")
    elif job == "update":
        runner = UpdateRunner(database_instance, "update")
    else:
        runner = ImportRunner(database_instance, "import")

    result = runner.run()
    if result is None:
        result = {}

    end = time.time()
    elapsed_time = end - start

    runner.store_report(job, elapsed_time, result)


# AWS Lambda
def handler(event, context):
    job = event["job"] if "job" in event else "import"
    start(job)


if __name__ == "__main__":
    job = sys.argv[1] if len(sys.argv) > 1 else "import"
    start(job)
