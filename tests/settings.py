# -*- coding: utf-8 -*-
"""Describe project settings"""
import os

BASEDIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DEPLOYMENTS = f"{BASEDIR}/deployments"
PROJECT_ROOT = os.path.abspath(os.path.dirname(__file__))
TEST_DATA = f"{PROJECT_ROOT}/data"
NUM_REPLICAS = 1
DEFAULT_IMAGE = "nginx/nginx-ingress:edge"
DEFAULT_PULL_POLICY = "IfNotPresent"
DEFAULT_IC_TYPE = "nginx-ingress"
ALLOWED_IC_TYPES = ["nginx-ingress", "nginx-plus-ingress"]
DEFAULT_SERVICE = "nodeport"
ALLOWED_SERVICE_TYPES = ["nodeport", "loadbalancer"]
DEFAULT_DEPLOYMENT_TYPE = "deployment"
ALLOWED_DEPLOYMENT_TYPES = ["deployment", "daemon-set"]
BATCH_START = "False"
# Number of Ingress/VS resources to deploy based on BATCH_START value, ref. line #264 in rresource_utils.py
BATCH_RESOURCES = 1
# Time in seconds to ensure reconfiguration changes in cluster
RECONFIGURATION_DELAY = 3
NGINX_API_VERSION = 4
