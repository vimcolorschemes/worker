import os
import pymongo
from bson.codec_options import CodecOptions

import printer

DATABASE_CONNECTION_STRING = os.getenv("DATABASE_CONNECTION_STRING")
if DATABASE_CONNECTION_STRING is None:
    DATABASE_CONNECTION_STRING = "mongodb://localhost:27017/"

client = pymongo.MongoClient(DATABASE_CONNECTION_STRING)
database = client["colorschemes"]
owner_collection = database["owners"]
repository_collection = database["repositories"]
import_collection = database["imports"].with_options(
    codec_options=CodecOptions(tz_aware=True)
)


def get_last_import_at():
    printer.break_line(2)
    printer.info("GET last import")
    printer.break_line(2)

    result = import_collection.find_one(sort=[("import_at", pymongo.DESCENDING)])
    last_import_at = result["import_at"] if result is not None else None
    return last_import_at


def create_import(import_data):
    printer.info("CREATE import")
    import_collection.insert_one(import_data)
    printer.break_line()


def is_repository_new(owner_name, name):
    repository = repository_collection.find_one({"owner.name": owner_name, "name": name})
    return repository == None, repository


def upsert_repository(repository_data):
    # TODO: When updating images, check that the repository's featured image is
    # still in the image set, if not: set it to null

    owner_name = repository_data["owner"]["name"]
    name = repository_data["name"]

    printer.info(f"UPSERT repository {owner_name}/{name}")

    result = repository_collection.update(
        {"owner.name": owner_name, "name": name}, {"$set": repository_data}, True,
    )
    inserted = "updatedExisting" not in result or result["updatedExisting"] == False
    printer.info(f"Repository was {'inserted' if inserted else 'updated'}")
    printer.break_line()
