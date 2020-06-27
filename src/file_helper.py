import base64
import os
import re

import request
from print_helper import colors


def decode_file_content(data):
    base64_bytes = data.encode("utf-8")
    bytes = base64.b64decode(base64_bytes)
    return bytes.decode("utf-8")


def find_images(file_content, max_image_count):
    image_url_regex = r"\b(https?:\/\/\S+(?:png|jpe?g|webp))\b"
    standard_image_urls = re.findall(image_url_regex, file_content)

    github_camo_url_regex = (
        r"\bhttps?:\/\/camo.githubusercontent.com(\/[0-9a-zA-Z]*)+\b"
    )
    github_camo_urls = re.findall(github_camo_url_regex, file_content)

    image_urls = standard_image_urls + github_camo_urls

    valid_images = []
    index = 0

    while len(valid_images) < max_image_count and index < len(image_urls):
        image_url = image_urls[index]
        image = request.download_image(image_url)
        if image is not None:
            valid_images.append(image)
        index = index + 1

    return valid_images


def read_file(path):
    return open(path, "r").read()


def write_file(path, data):
    with open(path, "w") as file:
        file.write(data)
