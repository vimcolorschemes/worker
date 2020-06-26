import dateutil.parser as dparser
from datetime import datetime

from request_helper import get, post

API_URL = "http://localhost:1337"


def get_last_import_at():
    print("\nGet last import")
    imports, used_cache = get(f"{API_URL}/imports", {"_sort": "created_at:DESC"})
    if len(imports) > 0:
        last_import = imports[0]
        last_import_at = last_import["created_at"]
        if last_import_at is not None:
            last_import_at = dparser.parse(last_import_at, fuzzy=True)
            print(f"Last import at: {last_import_at}")
            return last_import_at
        return None


def get_repository_by_github_id(id):
    print(f"\nGet repository by github_id {id}")
    repositories, used_cache = get(f"{API_URL}/repositories", {"github_id": id})
    repository = repositories[0] if len(repositories) > 0 else None
    return repository


def get_owner_by_name(name):
    print(f"\nGet owner by name {name}")
    owners, used_cache = get(f"{API_URL}/owners", {"name": name})
    owner = owners[0] if len(owners) > 0 else None
    return owner


def create_owner(owner_data):
    print(f"\nCreate owner")
    owner, used_cache = post(f"{API_URL}/owners", owner_data)
    return owner


def create_repository(repository_data):
    print("\nCreate repository")
    repository, used_cache = post(f"{API_URL}/repositories", repository_data)
    return repository


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
