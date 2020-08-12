import os
import sys
import time
from enum import Enum

from database import Database
from worker import Worker
import github
import printer


class Job(Enum):
    IMPORT = "import"
    CLEAN = "clean"
    SALVAGE = "salvage"


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

    job = event["job"] if "job" in event else None

    start = time.time()

    if job == Job.CLEAN:
        worker_instance.run_clean()
    elif job == Job.SALVAGE:
        worker_instance.run_salvage()
    else:
        worker_instance.run_import()

    end = time.time()
    elapsed_time = end - start

    printer.success(f"{job} finished.")
    printer.info(f"Elapsed time: {elapsed_time}")


if __name__ == "__main__":
    work = sys.argv[1] if len(sys.argv) > 1 else "import"
    handler({"job": work}, None)
