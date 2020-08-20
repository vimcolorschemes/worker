import os
import sys
import time
from enum import Enum

from database import Database
import github
import printer

from import_runner import ImportRunner
from clean_runner import CleanRunner


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

    job = event["job"] if "job" in event else None

    start = time.time()

    if job == "clean":
        runner = CleanRunner(database_instance, "clean")
    else:
        runner = ImportRunner(database_instance, "import")

    runner.run()

    end = time.time()
    elapsed_time = end - start

    runner.store_report(job, elapsed_time)

    printer.success(f"{job} finished.")
    printer.info(f"Elapsed time: {elapsed_time}")


if __name__ == "__main__":
    work = sys.argv[1] if len(sys.argv) > 1 else "import"
    handler({"job": work}, None)
