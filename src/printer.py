import sys
import time
import logging


class colors:
    SUCCESS = "\033[92m"
    ERROR = "\033[91m"
    WARNING = "\033[33m"
    NORMAL = "\033[0m"
    INFO = "\033[34m"


def info(message):
    logging.info(f"{colors.INFO}INFO: {colors.NORMAL}{message}")


def error(message, prefix=""):
    prefix = f"{prefix} " if prefix != "" else ""
    logging.error(f"{colors.ERROR}{prefix}ERROR: {message}{colors.NORMAL}")


def success(message):
    logging.info(f"{colors.SUCCESS}SUCCESS: {message}{colors.NORMAL}")


def warning(message):
    logging.warning(f"{colors.WARNING}WARNING: {message}{colors.NORMAL}")


def log(message):
    logging.log(f"LOG: {message}")


def break_line():
    logging.info("")


def start_sleeping(sleep_time):
    for i in range(1, sleep_time + 1):
        info(f"{i}/{sleep_time}")
        sys.stdout.write("\033[F")
        time.sleep(1)
    break_line()
