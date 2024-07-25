/*
Copyright 2024 Rancher Labs, Inc.

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

// Code generated by main. DO NOT EDIT.

package v1beta1

import (
	v1beta1 "github.com/kube-logging/logging-operator/pkg/sdk/logging/api/v1beta1"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/v3/pkg/schemes"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	schemes.Register(v1beta1.AddToScheme)
}

type Interface interface {
	ClusterFlow() ClusterFlowController
	ClusterOutput() ClusterOutputController
	Logging() LoggingController
}

func New(controllerFactory controller.SharedControllerFactory) Interface {
	return &version{
		controllerFactory: controllerFactory,
	}
}

type version struct {
	controllerFactory controller.SharedControllerFactory
}

func (c *version) ClusterFlow() ClusterFlowController {
	return NewClusterFlowController(schema.GroupVersionKind{Group: "logging.banzaicloud.io", Version: "v1beta1", Kind: "ClusterFlow"}, "clusterflows", true, c.controllerFactory)
}
func (c *version) ClusterOutput() ClusterOutputController {
	return NewClusterOutputController(schema.GroupVersionKind{Group: "logging.banzaicloud.io", Version: "v1beta1", Kind: "ClusterOutput"}, "clusteroutputs", true, c.controllerFactory)
}
func (c *version) Logging() LoggingController {
	return NewLoggingController(schema.GroupVersionKind{Group: "logging.banzaicloud.io", Version: "v1beta1", Kind: "Logging"}, "loggings", false, c.controllerFactory)
}
