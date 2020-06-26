import dateutil.parser as dparser
from datetime import datetime

from request_helper import get, post

API_URL = "http://localhost:1337"


def get_last_import_at():
    print("\n Get last import")
    imports, used_cache = get(f"{API_URL}/imports", {"_sort": "created_at:DESC"})
    if len(imports) > 0:
        last_import = imports[0]
        last_import_at = last_import["created_at"]
        if last_import_at is not None:
            last_import_at = dparser.parse(last_import_at, fuzzy=True)
            print(f"Last import at: {last_import_at}")
            return last_import_at
        return None


# TODO implement
def get_repository_by_github_id(id):
    print("get_repository_by_github_id")
    return None


# TODO implement
def insert_repository(repository):
    print("insert_repository")
    return {"id": 1}


# TODO implement
def update_repository(id, repository):
    print("update_repository")
    return {"id": 1}


# TODO implement
def upload_images(repository_id, image_urls):
    print(f"\nUpload images")


def create_import(import_data):
    print(f"\nCreate import")
    post(f"{API_URL}/imports", import_data)
