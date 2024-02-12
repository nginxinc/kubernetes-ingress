import re
import subprocess
from datetime import datetime

import requests


def collect_prom_reload_metrics(metric_list, scenario, ip, port) -> None:
    req_url = f"http://{ip}:{port}/metrics"
    resp = requests.get(req_url)
    resp_decoded = resp.content.decode("utf-8")
    reload_metric = ""
    for line in resp_decoded.splitlines():
        if "last_reload_milliseconds{class" in line:
            reload_metric = re.findall(r"\d+", line)[0]
            metric_list.append(
                {
                    f"Reload time ({scenario}) ": f"{reload_metric}ms",
                    "TimeStamp": str(datetime.utcnow()),
                }
            )


def run_perf(url, setup_users, setup_rate, setup_time, resource):
    subprocess.run(
        [
            "locust",
            "-f",
            f"suite/{resource}_request_perf.py",
            "--headless",
            "--host",
            url,
            "--csv",
            f"{resource}_response_times",
            "-u",
            setup_users,  # total no. of users
            "-r",
            setup_rate,  # no. of users hatched per second
            "-t",
            setup_time,  # locust session duration in seconds
        ]
    )
