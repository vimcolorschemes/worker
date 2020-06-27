import os
import base64
import requests
import requests_cache
from urllib import request

from print_helper import colors

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


def get(url, params={}, auth=None, is_json=True):
    used_cache = False
    try:
        response = requests.get(url, params=params, timeout=TIMEOUT, auth=auth)
        response.raise_for_status()
        if USE_CACHE:
            used_cache = response.from_cache
        data = response.json() if is_json else response
        return data, used_cache
    except requests.exceptions.HTTPError as errh:
        if response.status_code == 404:
            print(f"{colors.WARNING}Warning: 404 Not found for {url}{colors.NORMAL}")
            return None, used_cache
        print(f"{colors.ERROR}Http Error:{colors.NORMAL}", errh)
    except requests.exceptions.ConnectionError as errc:
        print(f"{colors.ERROR}Error Connecting:{colors.NORMAL}", errc)
    except requests.exceptions.Timeout as errt:
        print(f"{colors.ERROR}Timeout Error:{colors.NORMAL}", errt)
    except requests.exceptions.RequestException as err:
        print(f"{colors.ERROR}Request error:{colors.NORMAL}", err)
    except ValueError as errv:
        print(f"{colors.ERROR}Error decoding JSON response:{colors.NORMAL}", errv)
    except Exception as e:
        print(f"{colors.ERROR}Unexpected error{colors.NORMAL}:", e)
    return None, used_cache


def post(url, data=None, files=None, auth=None, is_json=True):
    used_cache = False
    try:
        response = requests.post(
            url, data=data, files=files, timeout=TIMEOUT, auth=auth
        )
        response.raise_for_status()
        if USE_CACHE:
            used_cache = response.from_cache
        data = response.json() if is_json else response
        return data, used_cache
    except requests.exceptions.HTTPError as errh:
        if response.status_code == 404:
            print(f"{colors.WARNING}Warning: 404 Not found for {url}{colors.NORMAL}")
            return None, used_cache
        print(f"{colors.ERROR}Http Error:{colors.NORMAL}", errh)
    except requests.exceptions.ConnectionError as errc:
        print(f"{colors.ERROR}Error Connecting:{colors.NORMAL}", errc)
    except requests.exceptions.Timeout as errt:
        print(f"{colors.ERROR}Timeout Error:{colors.NORMAL}", errt)
    except requests.exceptions.RequestException as err:
        print(f"{colors.ERROR}Request error:{colors.NORMAL}", err)
    except ValueError as errv:
        print(f"{colors.ERROR}Error decoding JSON response:{colors.NORMAL}", errv)
    except Exception as e:
        print(f"{colors.ERROR}Unexpected error{colors.NORMAL}:", e)
    return None, used_cache


def put(url, data={}, auth=None, is_json=True):
    used_cache = False
    try:
        response = requests.put(url, data=data, timeout=TIMEOUT, auth=auth)
        response.raise_for_status()
        if USE_CACHE:
            used_cache = response.from_cache
        data = response.json() if is_json else response
        return data, used_cache
    except requests.exceptions.HTTPError as errh:
        if response.status_code == 404:
            print(f"{colors.WARNING}Warning: 404 Not found for {url}{colors.NORMAL}")
            return None, used_cache
        print(f"{colors.ERROR}Http Error:{colors.NORMAL}", errh)
    except requests.exceptions.ConnectionError as errc:
        print(f"{colors.ERROR}Error Connecting:{colors.NORMAL}", errc)
    except requests.exceptions.Timeout as errt:
        print(f"{colors.ERROR}Timeout Error:{colors.NORMAL}", errt)
    except requests.exceptions.RequestException as err:
        print(f"{colors.ERROR}Request error:{colors.NORMAL}", err)
    except ValueError as errv:
        print(f"{colors.ERROR}Error decoding JSON response:{colors.NORMAL}", errv)
    except Exception as e:
        print(f"{colors.ERROR}Unexpected error{colors.NORMAL}:", e)
    return None, used_cache


def delete(url, auth=None, is_json=True):
    used_cache = False
    try:
        response = requests.delete(url, timeout=TIMEOUT, auth=auth)
        response.raise_for_status()
        if USE_CACHE:
            used_cache = response.from_cache
        data = response.json() if is_json else response
        return data, used_cache
    except requests.exceptions.HTTPError as errh:
        if response.status_code == 404:
            print(f"{colors.WARNING}Warning: 404 Not found for {url}{colors.NORMAL}")
            return None, used_cache
        print(f"{colors.ERROR}Http Error:{colors.NORMAL}", errh)
    except requests.exceptions.ConnectionError as errc:
        print(f"{colors.ERROR}Error Connecting:{colors.NORMAL}", errc)
    except requests.exceptions.Timeout as errt:
        print(f"{colors.ERROR}Timeout Error:{colors.NORMAL}", errt)
    except requests.exceptions.RequestException as err:
        print(f"{colors.ERROR}Request error:{colors.NORMAL}", err)
    except ValueError as errv:
        print(f"{colors.ERROR}Error decoding JSON response:{colors.NORMAL}", errv)
    except Exception as e:
        print(f"{colors.ERROR}Unexpected error{colors.NORMAL}:", e)
    return None, used_cache


def download_image(url):
    response, used_cache = get(url, is_json=False)
    print(
        f"{colors.INFO}GET{colors.NORMAL} image at url={url} (used_cache={used_cache})"
    )
    content_type = response.headers["Content-Type"]
    if response is not None and content_type in VALID_IMAGE_CONTENT_TYPES:
        return {
            "file_content": response.content,
            "content_type": content_type,
            "url": url,
        }
    return None
