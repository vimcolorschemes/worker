import dateutil.parser as dparser
import mimetypes
import os

import printer
import request

API_URL = os.getenv("API_URL")


def get_last_import_at():
    printer.info("GET last import")

    imports = request.get(f"{API_URL}/imports", {"_sort": "created_at:DESC"})
    if len(imports) > 0:
        last_import = imports[0]
        last_import_at = last_import["created_at"]
        if last_import_at is not None:
            last_import_at = dparser.parse(last_import_at, fuzzy=True)

            printer.info(f"Last import at {last_import_at}")

            return last_import_at

    return None


def get_repository_by_github_id(id):
    printer.info(f"GET repository with GitHub id: {id}")

    repositories = request.get(f"{API_URL}/repositories", {"github_id": id})

    repository = repositories[0] if len(repositories) > 0 else None

    printer.info(
        "Repository exists" if repository is not None else "Repository does not exist"
    )

    return repository


def get_owner_by_name(name):
    owners = request.get(f"{API_URL}/owners", {"name": name})
    owner = owners[0] if len(owners) > 0 else None
    return owner


def create_owner(owner_data):
    owner = request.post(f"{API_URL}/owners", owner_data)
    return owner


def create_repository(repository_data):
    repository = request.post(f"{API_URL}/repositories", repository_data)
    return repository


def update_repository(id, repository):
    printer.info("UPDATE repository")
    repository = request.put(f"{API_URL}/repositories/{id}", repository)
    return repository


def delete_images(images):
    for image in images:
        request.delete(f"{API_URL}/upload/files/{image['id']}")


def upload_repository_images(repository, images):
    for index, image in enumerate(images):
        printer.info(f"UPLOAD {image['url']}")
        repository_key = (
            f"{repository['owner']['name']}-{repository['name']}"
            if repository is not None and repository["owner"] is not None
            else "image"
        )
        file_extension = mimetypes.guess_extension(image["content_type"])
        if file_extension is None:
            file_extension = ".png"
        request.post(
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
    printer.info("CREATE import")
    request.post(f"{API_URL}/imports", import_data)
    printer.break_line()
