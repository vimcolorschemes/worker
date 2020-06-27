import dateutil.parser as dparser
import mimetypes
from datetime import datetime

from request_helper import get, post, put, download_image, delete

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


def update_repository(id, repository):
    print("\nUpdate repository")
    repository, used_cache = put(f"{API_URL}/repositories/{id}", repository)
    return repository


def delete_images(images):
    print(f"\nDelete images")

    for image in images:
        delete(f"{API_URL}/upload/files/{image['id']}")


def upload_repository_images(repository, images):
    print(f"\nUpload images")

    for index, image in enumerate(images):
        print(f"\nUploading {image['url']}")
        repository_key = (
            f"{repository['owner']['name']}-{repository['name']}"
            if repository is not None and repository["owner"] is not None
            else "image"
        )
        file_extension = mimetypes.guess_extension(image["content_type"])
        if file_extension is None:
            file_extension = ".png"
        post(
            f"{API_URL}/upload",
            files={
                "files": (
                    f"{repository_key}-{str(index + 1)}{file_extension}",
                    image["file_content"],
                    image["content_type"],
                ),
            },
            data={"ref": "repository", "field": "images", "refId": repository["id"],},
        )


def create_import(import_data):
    print(f"\nCreate import")
    post(f"{API_URL}/imports", import_data)
