import pytest, logging, io
from kubernetes.client.rest import ApiException
from suite.resources_utils import get_first_pod_name

@pytest.mark.ingresses
@pytest.mark.smoke
class TestBuildVersion:
    def test_build_version(
        self, ingress_controller, kube_apis, ingress_controller_prerequisites
    ):
        """
        Test Version tag of build i.e. 'Version=<VERSION>'
        """
        _info = self.send_build_info(kube_apis, ingress_controller_prerequisites)
        _version = _info[
            _info.find("Version=") + len("Version=") : _info.rfind("GitCommit=")
        ]
        logging.info(_version)
        assert _version != " "

    def test_build_gitcommit(
        self, ingress_controller, kube_apis, ingress_controller_prerequisites
    ):
        """
        Test Git Commit tag of build i.e. 'GitCommit=<GITCOMMIT>'
        """
        _info = self.send_build_info(kube_apis, ingress_controller_prerequisites)
        _commit = _info[_info.find("GitCommit=") :].lstrip().replace("GitCommit=", "")
        logging.info(_commit)
        assert _commit != ""

    def send_build_info(self, kube_apis, ingress_controller_prerequisites) -> str:
        """
        Helper function to get pod logs
        """
        pod_name = get_first_pod_name(
            kube_apis.v1, ingress_controller_prerequisites.namespace
        )
        try:
            api_response = kube_apis.v1.read_namespaced_pod_log(
                name=pod_name,
                namespace=ingress_controller_prerequisites.namespace,
                limit_bytes=200,
            )
            logging.info(api_response)
        except ApiException as e:
            logging.exception(f"Found exception in reading the logs: {e}")

        br = io.StringIO(api_response)
        _log = br.readline()
        try:
            _info = _log[_log.find("Version") :].strip()
            logging.info(f"Version and GitCommit info: {_info}")
        except Exception as e:
            logging.exception(f"Tag labels not found")

        return _info
