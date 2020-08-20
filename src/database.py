import os
import pymongo
import datetime
from bson.codec_options import CodecOptions

import printer


class Database:
    def __init__(self, host, username="", password=""):
        self.client = pymongo.MongoClient(
            host=host, username=username, password=password
        )
        self.database = self.client["colorschemes"]
        self.repository_collection = self.database["repositories"]
        self.report_collection = self.database["reports"].with_options(
            codec_options=CodecOptions(tz_aware=True)
        )

    def get_last_job_at(self, job):
        printer.break_line(2)
        printer.info("GET last import")
        printer.break_line(2)

        result = self.report_collection.find_one(
            {"job": job}, sort=[("created_at", pymongo.DESCENDING)]
        )
        last_job_at = result["created_at"] if result is not None else None
        return last_job_at

    def create_report(self, report_data):
        printer.info("CREATE report")
        self.report_collection.insert_one(
            {**report_data, "created_at": datetime.datetime.now()}
        )
        printer.break_line()

    def upsert_repository(self, repository_data):
        # TODO: When updating images, check that the repository's featured image is
        # still in the image set, if not: set it to null

        owner_name = repository_data["owner"]["name"]
        name = repository_data["name"]

        printer.info(f"UPSERT repository {owner_name}/{name}")

        result = self.repository_collection.update(
            {"owner.name": owner_name, "name": name}, {"$set": repository_data}, True,
        )
        inserted = "updatedExisting" not in result or result["updatedExisting"] == False
        printer.info(f"Repository was {'inserted' if inserted else 'updated'}")
        printer.break_line()

    def get_repository(self, owner_name, name):
        return self.repository_collection.find_one(
            {"owner.name": owner_name, "name": name}
        )

    def get_repositories(self):
        result = self.repository_collection.find()
        return list(result)
