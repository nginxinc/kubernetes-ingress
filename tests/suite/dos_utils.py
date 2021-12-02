from suite.resources_utils import get_file_contents, wait_before_test


def log_content_to_dic(log_contents):
    arr = []
    for line in log_contents.splitlines():
        if line.__contains__('app-protect-dos'):
            arr.append(line)

    log_info_dic = []
    for line in arr:
        chunks = line.split(",")
        d = {}
        for chunk in chunks:
            tmp = chunk.split("=")
            if len(tmp) == 2:
                if 'date_time' in tmp[0]:
                    tmp[0] = 'date_time'
                d[tmp[0].strip()] = tmp[1].replace('"', '')
        log_info_dic.append(d)
    return log_info_dic


def find_in_log(kube_apis, log_location, syslog_pod, namespace, time, value):
    log_contents = ""
    retry = 0
    while (
            value not in log_contents
            and retry <= time / 10
    ):
        log_contents = get_file_contents(kube_apis.v1, log_location, syslog_pod, namespace, False)
        retry += 1
        wait_before_test(10)
        print(f"{value} Not in log, retrying... #{retry}")
