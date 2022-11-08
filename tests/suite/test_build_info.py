import io
import logging
import time

import pytest
from suite.utils.resources_utils import get_first_pod_name, wait_until_all_pods_are_ready


@pytest.mark.ingresses
@pytest.mark.smoke
class TestBuildVersion:
    def test_build_version(self, ingress_controller, kube_apis, ingress_controller_prerequisites):
        """
        Test Version tag of build i.e. 'Version=<VERSION>'
        """
        _info = self.send_build_info(kube_apis, ingress_controller_prerequisites)
        _version = _info[_info.find("Version=") + len("Version=") : _info.rfind("GitCommit=")]
        logging.info(_version)
        assert _version != " "

    def send_build_info(self, kube_apis, ingress_controller_prerequisites) -> str:
        """
        Helper function to get pod logs
        """
        retry = 0
        ready = False
        pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        wait_until_all_pods_are_ready(kube_apis.v1, ingress_controller_prerequisites.namespace)
        while not ready:
            time.sleep(1)
            try:
                api_response = kube_apis.v1.read_namespaced_pod_log(
                    name=pod_name,
                    namespace=ingress_controller_prerequisites.namespace,
                    limit_bytes=200,
                )
                logging.info(api_response)
                ready = True

            except Exception as ex:
                if retry < 10:
                    retry += 1
                    print(f"Retry# {retry}")
                else:
                    logging.exception(ex)
                    raise ex

        br = io.StringIO(api_response)
        _log = br.readline().strip()
        try:
            _info = _log[_log.find("Version") :].strip()
            logging.info(f"Version and GitCommit info: {_info}")
        except Exception as e:
            logging.exception(f"Tag labels not found")

        return _info
