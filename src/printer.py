import sys
import time


class colors:
    SUCCESS = "\033[92m"
    ERROR = "\033[91m"
    WARNING = "\033[33m"
    NORMAL = "\033[0m"
    INFO = "\033[34m"


def info(message):
    print(f"{colors.INFO}INFO: {colors.NORMAL}{message}")


def error(message, prefix=""):
    prefix = f"{prefix} " if prefix != "" else ""
    print(f"{colors.ERROR}{prefix}ERROR: {message}{colors.NORMAL}")


def success(message):
    print(f"{colors.SUCCESS}SUCCESS: {message}{colors.NORMAL}")


def warning(message):
    print(f"{colors.WARNING}WARNING: {message}{colors.NORMAL}")


def log(message):
    print(f"LOG: {message}")


def break_line(n = 1):
    for i in range(0, n):
        print("")


def start_sleeping(sleep_time):
    for i in range(1, sleep_time + 1):
        info(f"{i}/{sleep_time}")
        sys.stdout.write("\033[F")
        time.sleep(1)
    break_line()
    break_line()
