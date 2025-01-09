import os
import subprocess

from kubernetes.client import CoreV1Api
from kubernetes.stream import stream
from suite.utils.resources_utils import get_file_contents, wait_before_test


def log_content_to_dic(log_contents):
    arr = []
    for line in log_contents.splitlines():
        if line.__contains__("app-protect-dos"):
            arr.append(line)

    log_info_dic = []
    for line in arr:
        chunks = line.split(",")
        d = {}
        for chunk in chunks:
            tmp = chunk.split("=")
            if len(tmp) == 2:
                if "date_time" in tmp[0]:
                    tmp[0] = "date_time"
                d[tmp[0].strip()] = tmp[1].replace('"', "")
        log_info_dic.append(d)
    return log_info_dic


def find_in_log(kube_apis, log_location, syslog_pod, namespace, time, value):
    log_contents = ""
    retry = 0
    while value not in log_contents and retry <= time / 10:
        log_contents = get_file_contents(kube_apis.v1, log_location, syslog_pod, namespace, False)
        retry += 1
        wait_before_test(10)
        print(f"{value} Not in log, retrying... #{retry}")


def admd_s_content_to_dic(admd_s_contents):
    arr = []
    for line in admd_s_contents.splitlines():
        arr.append(line)

    admd_s_dic = {}
    for line in arr:
        tmp = line.split(":")
        admd_s_dic[tmp[0].split("/")[-1]] = tmp[1]
    return admd_s_dic


def check_learning_status_with_admd_s(kube_apis, syslog_pod, namespace, time):
    retry = 0
    learning_sas = 0.0
    learning_signature = 0.0
    retry_time = 15
    while (learning_sas < 75 or learning_signature != 100) and retry <= time / retry_time:
        retry += 1
        admd_contents = get_admd_s_contents(kube_apis.v1, syslog_pod, namespace, retry_time)
        admd_s_dic = admd_s_content_to_dic(admd_contents)
        if "name.info.learning" not in admd_s_dic:
            print("name.info.learning not found in admd_s_dic")
            wait_before_test(retry_time)
            continue
        learn = admd_s_dic["name.info.learning"].replace("[", "").replace("]", "").split(",")
        learning_sas = float(learn[0])
        learning_signature = float(learn[3])
        print(f"learning_sas: {learning_sas}, learning_signature: {learning_signature}")


def get_admd_s_contents(v1: CoreV1Api, pod_name, pod_namespace, time):
    command = ["admd", "-s", "vs."]
    resp = stream(
        v1.connect_get_namespaced_pod_exec,
        pod_name,
        pod_namespace,
        command=command,
        stderr=True,
        stdin=False,
        stdout=True,
        tty=False,
        _request_timeout=time,
    )
    admd_contents = str(resp)
    return admd_contents


def clean_good_bad_clients():
    command = "exec ps -aux | grep good_clients_xff.sh | awk '{print $2}' | xargs kill -9"

    subprocess.Popen(
        [command],
        preexec_fn=os.setsid,
        shell=True,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )

    command = "exec ps -aux | grep bad_clients_xff.sh | awk '{print $2}' | xargs kill -9"
    subprocess.Popen(
        [command],
        preexec_fn=os.setsid,
        shell=True,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )


def print_admd_log(log):
    matches = ["ADMD", "DAEMONLESS"]
    for line in log.splitlines():
        if any(x in line for x in matches):
            print(line)
