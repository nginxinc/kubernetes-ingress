---
title: Using the AWS Marketplace Ingress Controller Image
description: "This document will walk you through the steps needed to use the NGINX Ingress Controller through the AWS Marketplace."
weight: 2300
doctypes: [""]
toc: true
docs: "DOCS-607"
---

This document will walk you through the steps needed to use the NGINX Ingress Controller through the AWS Marketplace. There are additional steps that must be followed in order for the AWS Marketplace NGINX Ingress Controller to work properly.

> **IMPORTANT**: This document uses EKS version 1.19. EKS versions < 1.19 require additional security settings within the NGINX Pod to work properly with marketplace images.
> This document discusses using eksctl to perform necessary steps to enable the Kubernetes cluster access to deploy NGINX Ingress Controller from the Marketplace. Please make sure you are running a newer version of eksctl and AWS cli.

> **NOTE**: NGINX Ingress Controller from the Marketplace does NOT work in AWS Region US-West-1.

## Instructions
Instructions for using AWS Marketplace:

1. Ensure you have a working AWS EKS cluster. If you do not have a EKS cluster, you can create one using either the AWS console, or using the AWS tool eksctl. See [this guide](https://docs.aws.amazon.com/eks/latest/userguide/getting-started-eksctl.html) for details on getting started with EKS using eksctl.

2. You must create a new IAM role that will be associated with the Service Account created for the NGINX Ingress Controller. This IAM role will have a specific IAM policy that allows you to monitor the usage of the AWS NGINX Ingress Controller image. This is a required step. If it is omitted, AWS Marketplace NGINX Ingress Controller will not work properly and NGINX Ingress will not start. Please see [the AWS EKS IAM documentation here](https://docs.aws.amazon.com/eks/latest/userguide/create-service-account-iam-policy-and-role.html) and the documentation detailing the policy required for the [AWS Marketplace here](https://docs.aws.amazon.com/marketplace/latest/userguide/iam-user-policy-for-aws-marketplace-actions.html).

3. You must associate this IAM role with your service account in your EKS cluster. When you do so, your service account Kubernetes object will have a annotation, showing the link to the IAM role.


> **NB** You must associate your AWS EKS cluster with an OIDC provider before you can create your IAM Service account! This is required.

## Step by step instructions using eksctl utility.

This assumes you have an existing EKS cluster up and running. If not, please create one before proceeding. This also assumes the namespace for the NGINX Ingress Controller already exists.

1. Associate your EKS cluster with a “OIDC IAM provider” (replace `--cluster <name>` and `--region <region>` with the values of your environment).
```
eksctl utils associate-iam-oidc-provider --region=eu-west-1 --cluster=json-eu-east1 --approve
```

2.  Now create your IAM role and service account for your cluster. Substitute `--name <name>` `--namespace <name>` and `--region <region>` with your values.
```
eksctl create iamserviceaccount --name nginx-ingress --namespace nginx-ingress --cluster json-test01 --region us-east-2 --attach-policy-arn arn:aws:iam::aws:policy/AWSMarketplaceMeteringRegisterUsage --approve
 ```

This will create the IAM role with the required policy attached, create the service account if it doesn't already exist, and add the annotations needed for your AWS cluster. See [the documentation here](https://docs.aws.amazon.com/eks/latest/userguide/create-service-account-iam-policy-and-role.html). Since eksctl is creating it for you, you do not need to apply any service account yaml files for your NGINX Ingress Controller deployments.

```
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    EKS.amazonaws.com/role-arn: arn:aws:iam::001234567890:role/eksctl-json-us-west2-addon-iamserviceaccount-Role1-IJJ6CF9Y8IPY
  labels:
    app.kubernetes.io/managed-by: eksctl
  name: nginx-ingress
  namespace: nginx-ingress
secrets:
- name: nginx-ingress-token-zm728
```

Make sure you match the name you are creating for the service account, to the account that will be in the `rbac.yaml` file for manifest deployment.

Sample output from the `rbac.yaml` file, matching the IAM service account that was created above:

```
kind: ClusterRoleBinding
  apiVersion: rbac.authorization.k8s.io/v1
  metadata:
    name: nginx-ingress
  subjects:
  - kind: ServiceAccount
    name: nginx-ingress
    namespace: nginx-ingress
  roleRef:
    kind: ClusterRole
    name: nginx-ingress
    apiGroup: rbac.authorization.k8s.io
```

3. Log into the AWS ECR registry that is specified in the instructions from the [AWS Marketplace portal](https://aws.amazon.com/marketplace/pp/prodview-fx3faxl7zqeau?sr=0-1&ref_=beagle&applicationId=AWSMPContessa).

{{< img title="ECR Pull Instructions for the NGINX Ingress Controller" src="./img/ecr-pull-instructions.png" >}}

**Note:** AWS Labs also provides a credential helper - see [their GitHub repo](https://github.com/awslabs/amazon-ecr-credential-helper) for instructions on how to setup and configure.

4. Update the image in the `nginx-plus-ingress.yaml` manifest.
