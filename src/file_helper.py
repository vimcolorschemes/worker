import base64
import os
import re

import request


def find_images(file_content, max_image_count):
    image_url_regex = r"\b(https?:\/\/\S+(?:png|jpe?g|webp))\b"
    standard_image_urls = re.findall(image_url_regex, file_content)

    github_camo_url_regex = (
        r"\bhttps?:\/\/camo.githubusercontent.com(\/[0-9a-zA-Z]*)+\b"
    )
    github_camo_urls = re.findall(github_camo_url_regex, file_content)

    image_urls = standard_image_urls + github_camo_urls

    valid_image_urls = []
    index = 0

    while len(valid_image_urls) < max_image_count and index < len(image_urls):
        image_url = image_urls[index]
        if request.is_image_url_valid(image_url):
            valid_image_urls.append(image_url)
        index = index + 1

    return valid_image_urls


def read_file(path):
    return open(path, "r").read()


def write_file(path, data):
    with open(path, "w") as file:
        file.write(data)
