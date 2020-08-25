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


def find_image_urls(file_content, max_image_count):
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


def build_raw_blog_github_url(owner_name, name, path):
    return f"https://raw.githubusercontent.com/{owner_name}/{name}/{path}"


def get_vim_color_scheme_name(owner_name, name, file):
    vim_color_scheme_name = None
    response = request.get(
        build_raw_blog_github_url(owner_name, name, file["path"]), is_json=False
    )
    file_content = response.text if response is not None else ""

    match = re.search(r"let (g:)?colors?_name ?= ?('|\")([a-zA-Z-_0-9]+)('|\")", file_content)
    if match is not None:
        vim_color_scheme_name = match.group(3)
        printer.info(f"{name} vim color scheme name is {vim_color_scheme_name}")

    return vim_color_scheme_name
