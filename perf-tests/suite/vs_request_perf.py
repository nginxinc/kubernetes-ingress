import os

import yaml
from locust import HttpUser, task

host = ""


class TestAPResponse(HttpUser):
    # locust class to be invoked
    def on_start(self):
        # get host from appprotect-ingress yaml before each test
        ing_yaml = os.path.join(
            os.path.dirname(__file__), "../../tests/data/virtual-server/standard/virtual-server.yaml"
        )
        with open(ing_yaml) as f:
            docs = yaml.safe_load_all(f)
            for dep in docs:
                self.host = dep["spec"]["host"]
        print("Setup finished")

    @task
    def send_request(self):
        response = self.client.get(url="", headers={"host": self.host}, verify=False)
        print(response.text)

    min_wait = 400
    max_wait = 1400
