import os
import base64
import requests
import requests_cache

import printer

TIMEOUT = 5

USE_CACHE = os.getenv("USE_CACHE")
CACHE_EXPIRE_AFTER = os.getenv("CACHE_EXPIRE_AFTER")

VALID_IMAGE_CONTENT_TYPES = ["image/jpeg", "image/png", "image/webp"]

if USE_CACHE:
    requests_cache.install_cache(
        "github_cache",
        backend="sqlite",
        expire_after=int(CACHE_EXPIRE_AFTER)
        if CACHE_EXPIRE_AFTER is not None
        else 3600,
    )


def request(verb, url, params=None, data=None, files=None, auth=None, is_json=True):
    used_cache = False
    try:
        action = getattr(requests, verb, None)
        if not action:
            printer.error(f"Wrong verb for request module: {verb}")
            return None, used_cache

        response = action(
            url=url, params=params, data=data, files=files, auth=auth, timeout=TIMEOUT
        )
        response.raise_for_status()

        if USE_CACHE:
            printer.info(f"Used cache")
            used_cache = response.from_cache


        data = response.json() if is_json else response
        return data, used_cache
    except requests.exceptions.HTTPError as errh:
        if response.status_code == 404:
            printer.warning(f"404 Not found for {url}")
            return None, used_cache
        printer.error(errh, "HTTP")
    except requests.exceptions.ConnectionError as errc:
        printer.error(errc, "CONNECTION")
    except requests.exceptions.Timeout as errt:
        printer.error(errt, "TIMEOUT")
    except requests.exceptions.RequestException as err:
        printer.error(err, "REQUEST")
    except ValueError as errv:
        printer.error(errv, "VALUE ERROR")
    except Exception as e:
        printer.error(e, "UNEXPECTED ERROR")
    return None, used_cache


def get(url, params={}, auth=None, is_json=True):
    return request(verb="get", url=url, params=params, auth=auth, is_json=is_json)


def post(url, data=None, files=None, auth=None, is_json=True):
    return request(
        verb="post", url=url, data=data, files=files, auth=auth, is_json=is_json
    )


def put(url, data={}, auth=None, is_json=True):
    return request(verb="put", url=url, data=data, auth=auth, is_json=is_json)


def delete(url, auth=None):
    return request(verb="delete", url=url, auth=auth, is_json=False)


def download_image(url):
    printer.info(f"DOWNLOAD image at url {url}")
    response, used_cache = get(url, is_json=False)
    if response is not None:
        content_type = response.headers["Content-Type"]
        if content_type in VALID_IMAGE_CONTENT_TYPES:
            return {
                "file_content": response.content,
                "content_type": content_type,
                "url": url,
            }
    return None
