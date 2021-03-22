A linter to enforce importing certain packages consistently.

## What is this for?

Ideally, go imports should avoid aliasing. Sometimes though, especially with
Kubernetes API code, it becomes unavoidable, because many packages are imported
as e.g. "[package]/v1alpha1" and you end up with lots of collisions if you use
"v1alpha1". 

This linter lets you enforce that whenever (for example)
"pkg/apis/serving/v1alpha1" is aliased, it is aliased as "servingv1alpha1".

## Usage

~~~~
importas \
  -alias knative.dev/serving/pkg/apis/autoscaling/v1alpha1:autoscalingv1alpha1 \
  -alias knative.dev/serving/pkg/apis/serving/v1:servingv1 \
  ./...
~~~~
