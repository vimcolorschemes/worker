import base64
import requests

import printer

TIMEOUT = 15


def request(verb, url, params=None, data=None, files=None, auth=None, is_json=True):
    try:
        action = getattr(requests, verb, None)
        if not action:
            printer.error(f"Wrong verb for request module: {verb}")
            return None

        response = action(
            url=url, params=params, data=data, files=files, auth=auth, timeout=TIMEOUT
        )
        response.raise_for_status()

        data = response.json() if is_json else response
        return data
    except requests.exceptions.HTTPError as errh:
        if response.status_code == 404:
            printer.warning(f"404 Not found for {url}")
            return None
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
    return None


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


def is_url_valid(url, allow_redirects=True):
    try:
        printer.info(f"HEAD at url {url}")
        response = requests.head(url, allow_redirects=allow_redirects)
        if response.status_code == 200:
            return True
        return False
    except Exception as e:
        printer.error(e)
        return False


VALID_IMAGE_CONTENT_TYPES = ["image/jpeg", "image/png", "image/webp"]


def is_image_url_valid(url):
    try:
        printer.info(f"HEAD image at url {url}")
        response = requests.head(url, allow_redirects=True)
        if response.status_code == 200:
            content_type = response.headers["Content-Type"]
            if content_type in VALID_IMAGE_CONTENT_TYPES:
                return True
        return False
    except Exception as e:
        printer.error(e)
        return False
