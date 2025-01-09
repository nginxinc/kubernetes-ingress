/*
Copyright 2020 The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package test

import (
	"context"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"

	clientset "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	informers "github.com/cert-manager/cert-manager/pkg/client/informers/externalversions"
)

// Context contains various types that are used by controller implementations.
type Context struct {
	// RootContext is the root context for the controller
	RootContext context.Context

	StopCh <-chan struct{}
	// RESTConfig is the loaded Kubernetes apiserver rest client configuration
	RESTConfig *rest.Config
	Client     kubernetes.Interface
	CMClient   clientset.Interface

	Recorder record.EventRecorder

	KubeSharedInformerFactory kubeinformers.SharedInformerFactory
	SharedInformerFactory     informers.SharedInformerFactory

	ContextOptions
}

// ContextOptions are static Controller Context options.
type ContextOptions struct {
	Kubeconfig string
	Namespace  string
	Clock      clock.Clock
}
