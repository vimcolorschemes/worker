import re
import base64

import request
import printer
import github


def decode_base64(data):
    try:
        base64_bytes = data.encode("utf-8")
        bytes = base64.b64decode(base64_bytes)
        return bytes.decode("utf-8")
    except Exception as e:
        printer.error(e)
        return ""


def find_image_urls(file_content):
    image_url_regex = r"\b(https?:\/\/\S+(?:png|jpe?g|webp))\b"
    standard_image_urls = re.findall(image_url_regex, file_content)

    github_camo_url_regex = (
        r"\bhttps?:\/\/camo.githubusercontent.com(\/[0-9a-zA-Z]*)+\b"
    )
    github_camo_urls = re.findall(github_camo_url_regex, file_content)

    image_urls = standard_image_urls + github_camo_urls

    valid_image_urls = []
    index = 0

    while index < len(image_urls):
        image_url = image_urls[index]
        if request.is_image_url_valid(image_url):
            valid_image_urls.append(image_url)
        index = index + 1

    return valid_image_urls


def build_raw_blog_github_url(owner_name, name, path):
    return f"https://raw.githubusercontent.com/{owner_name}/{name}/{path}"


def is_vim_color_scheme(owner_name, name, file):
    response = request.get(
        build_raw_blog_github_url(owner_name, name, file["path"]), is_json=False
    )
    file_content = response.text if response is not None else ""
    return "colors_name" in file_content
