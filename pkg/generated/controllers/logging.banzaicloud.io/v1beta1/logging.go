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
	"context"
	"sync"
	"time"

	v1beta1 "github.com/kube-logging/logging-operator/pkg/sdk/logging/api/v1beta1"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/v3/pkg/apply"
	"github.com/rancher/wrangler/v3/pkg/condition"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/rancher/wrangler/v3/pkg/kv"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type LoggingHandler func(string, *v1beta1.Logging) (*v1beta1.Logging, error)

type LoggingController interface {
	generic.ControllerMeta
	LoggingClient

	OnChange(ctx context.Context, name string, sync LoggingHandler)
	OnRemove(ctx context.Context, name string, sync LoggingHandler)
	Enqueue(name string)
	EnqueueAfter(name string, duration time.Duration)

	Cache() LoggingCache
}

type LoggingClient interface {
	Create(*v1beta1.Logging) (*v1beta1.Logging, error)
	Update(*v1beta1.Logging) (*v1beta1.Logging, error)
	UpdateStatus(*v1beta1.Logging) (*v1beta1.Logging, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Get(name string, options metav1.GetOptions) (*v1beta1.Logging, error)
	List(opts metav1.ListOptions) (*v1beta1.LoggingList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.Logging, err error)
}

type LoggingCache interface {
	Get(name string) (*v1beta1.Logging, error)
	List(selector labels.Selector) ([]*v1beta1.Logging, error)

	AddIndexer(indexName string, indexer LoggingIndexer)
	GetByIndex(indexName, key string) ([]*v1beta1.Logging, error)
}

type LoggingIndexer func(obj *v1beta1.Logging) ([]string, error)

type loggingController struct {
	controller    controller.SharedController
	client        *client.Client
	gvk           schema.GroupVersionKind
	groupResource schema.GroupResource
}

func NewLoggingController(gvk schema.GroupVersionKind, resource string, namespaced bool, controller controller.SharedControllerFactory) LoggingController {
	c := controller.ForResourceKind(gvk.GroupVersion().WithResource(resource), gvk.Kind, namespaced)
	return &loggingController{
		controller: c,
		client:     c.Client(),
		gvk:        gvk,
		groupResource: schema.GroupResource{
			Group:    gvk.Group,
			Resource: resource,
		},
	}
}

func FromLoggingHandlerToHandler(sync LoggingHandler) generic.Handler {
	return func(key string, obj runtime.Object) (ret runtime.Object, err error) {
		var v *v1beta1.Logging
		if obj == nil {
			v, err = sync(key, nil)
		} else {
			v, err = sync(key, obj.(*v1beta1.Logging))
		}
		if v == nil {
			return nil, err
		}
		return v, err
	}
}

func (c *loggingController) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Update(obj.(*v1beta1.Logging))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func UpdateLoggingDeepCopyOnChange(client LoggingClient, obj *v1beta1.Logging, handler func(obj *v1beta1.Logging) (*v1beta1.Logging, error)) (*v1beta1.Logging, error) {
	if obj == nil {
		return obj, nil
	}

	copyObj := obj.DeepCopy()
	newObj, err := handler(copyObj)
	if newObj != nil {
		copyObj = newObj
	}
	if obj.ResourceVersion == copyObj.ResourceVersion && !equality.Semantic.DeepEqual(obj, copyObj) {
		return client.Update(copyObj)
	}

	return copyObj, err
}

func (c *loggingController) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controller.RegisterHandler(ctx, name, controller.SharedControllerHandlerFunc(handler))
}

func (c *loggingController) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), handler))
}

func (c *loggingController) OnChange(ctx context.Context, name string, sync LoggingHandler) {
	c.AddGenericHandler(ctx, name, FromLoggingHandlerToHandler(sync))
}

func (c *loggingController) OnRemove(ctx context.Context, name string, sync LoggingHandler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), FromLoggingHandlerToHandler(sync)))
}

func (c *loggingController) Enqueue(name string) {
	c.controller.Enqueue("", name)
}

func (c *loggingController) EnqueueAfter(name string, duration time.Duration) {
	c.controller.EnqueueAfter("", name, duration)
}

func (c *loggingController) Informer() cache.SharedIndexInformer {
	return c.controller.Informer()
}

func (c *loggingController) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *loggingController) Cache() LoggingCache {
	return &loggingCache{
		indexer:  c.Informer().GetIndexer(),
		resource: c.groupResource,
	}
}

func (c *loggingController) Create(obj *v1beta1.Logging) (*v1beta1.Logging, error) {
	result := &v1beta1.Logging{}
	return result, c.client.Create(context.TODO(), "", obj, result, metav1.CreateOptions{})
}

func (c *loggingController) Update(obj *v1beta1.Logging) (*v1beta1.Logging, error) {
	result := &v1beta1.Logging{}
	return result, c.client.Update(context.TODO(), "", obj, result, metav1.UpdateOptions{})
}

func (c *loggingController) UpdateStatus(obj *v1beta1.Logging) (*v1beta1.Logging, error) {
	result := &v1beta1.Logging{}
	return result, c.client.UpdateStatus(context.TODO(), "", obj, result, metav1.UpdateOptions{})
}

func (c *loggingController) Delete(name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return c.client.Delete(context.TODO(), "", name, *options)
}

func (c *loggingController) Get(name string, options metav1.GetOptions) (*v1beta1.Logging, error) {
	result := &v1beta1.Logging{}
	return result, c.client.Get(context.TODO(), "", name, result, options)
}

func (c *loggingController) List(opts metav1.ListOptions) (*v1beta1.LoggingList, error) {
	result := &v1beta1.LoggingList{}
	return result, c.client.List(context.TODO(), "", result, opts)
}

func (c *loggingController) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch(context.TODO(), "", opts)
}

func (c *loggingController) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (*v1beta1.Logging, error) {
	result := &v1beta1.Logging{}
	return result, c.client.Patch(context.TODO(), "", name, pt, data, result, metav1.PatchOptions{}, subresources...)
}

type loggingCache struct {
	indexer  cache.Indexer
	resource schema.GroupResource
}

func (c *loggingCache) Get(name string) (*v1beta1.Logging, error) {
	obj, exists, err := c.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(c.resource, name)
	}
	return obj.(*v1beta1.Logging), nil
}

func (c *loggingCache) List(selector labels.Selector) (ret []*v1beta1.Logging, err error) {

	err = cache.ListAll(c.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1beta1.Logging))
	})

	return ret, err
}

func (c *loggingCache) AddIndexer(indexName string, indexer LoggingIndexer) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(*v1beta1.Logging))
		},
	}))
}

func (c *loggingCache) GetByIndex(indexName, key string) (result []*v1beta1.Logging, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]*v1beta1.Logging, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(*v1beta1.Logging))
	}
	return result, nil
}

// LoggingStatusHandler is executed for every added or modified Logging. Should return the new status to be updated
type LoggingStatusHandler func(obj *v1beta1.Logging, status v1beta1.LoggingStatus) (v1beta1.LoggingStatus, error)

// LoggingGeneratingHandler is the top-level handler that is executed for every Logging event. It extends LoggingStatusHandler by a returning a slice of child objects to be passed to apply.Apply
type LoggingGeneratingHandler func(obj *v1beta1.Logging, status v1beta1.LoggingStatus) ([]runtime.Object, v1beta1.LoggingStatus, error)

// RegisterLoggingStatusHandler configures a LoggingController to execute a LoggingStatusHandler for every events observed.
// If a non-empty condition is provided, it will be updated in the status conditions for every handler execution
func RegisterLoggingStatusHandler(ctx context.Context, controller LoggingController, condition condition.Cond, name string, handler LoggingStatusHandler) {
	statusHandler := &loggingStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, FromLoggingHandlerToHandler(statusHandler.sync))
}

// RegisterLoggingGeneratingHandler configures a LoggingController to execute a LoggingGeneratingHandler for every events observed, passing the returned objects to the provided apply.Apply.
// If a non-empty condition is provided, it will be updated in the status conditions for every handler execution
func RegisterLoggingGeneratingHandler(ctx context.Context, controller LoggingController, apply apply.Apply,
	condition condition.Cond, name string, handler LoggingGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &loggingGeneratingHandler{
		LoggingGeneratingHandler: handler,
		apply:                    apply,
		name:                     name,
		gvk:                      controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterLoggingStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type loggingStatusHandler struct {
	client    LoggingClient
	condition condition.Cond
	handler   LoggingStatusHandler
}

// sync is executed on every resource addition or modification. Executes the configured handlers and sends the updated status to the Kubernetes API
func (a *loggingStatusHandler) sync(key string, obj *v1beta1.Logging) (*v1beta1.Logging, error) {
	if obj == nil {
		return obj, nil
	}

	origStatus := obj.Status.DeepCopy()
	obj = obj.DeepCopy()
	newStatus, err := a.handler(obj, obj.Status)
	if err != nil {
		// Revert to old status on error
		newStatus = *origStatus.DeepCopy()
	}

	if a.condition != "" {
		if errors.IsConflict(err) {
			a.condition.SetError(&newStatus, "", nil)
		} else {
			a.condition.SetError(&newStatus, "", err)
		}
	}
	if !equality.Semantic.DeepEqual(origStatus, &newStatus) {
		if a.condition != "" {
			// Since status has changed, update the lastUpdatedTime
			a.condition.LastUpdated(&newStatus, time.Now().UTC().Format(time.RFC3339))
		}

		var newErr error
		obj.Status = newStatus
		newObj, newErr := a.client.UpdateStatus(obj)
		if err == nil {
			err = newErr
		}
		if newErr == nil {
			obj = newObj
		}
	}
	return obj, err
}

type loggingGeneratingHandler struct {
	LoggingGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
	seen  sync.Map
}

// Remove handles the observed deletion of a resource, cascade deleting every associated resource previously applied
func (a *loggingGeneratingHandler) Remove(key string, obj *v1beta1.Logging) (*v1beta1.Logging, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v1beta1.Logging{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	if a.opts.UniqueApplyForResourceVersion {
		a.seen.Delete(key)
	}

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

// Handle executes the configured LoggingGeneratingHandler and pass the resulting objects to apply.Apply, finally returning the new status of the resource
func (a *loggingGeneratingHandler) Handle(obj *v1beta1.Logging, status v1beta1.LoggingStatus) (v1beta1.LoggingStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.LoggingGeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}
	if !a.isNewResourceVersion(obj) {
		return newStatus, nil
	}

	err = generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
	if err != nil {
		return newStatus, err
	}
	a.storeResourceVersion(obj)
	return newStatus, nil
}

// isNewResourceVersion detects if a specific resource version was already successfully processed.
// Only used if UniqueApplyForResourceVersion is set in generic.GeneratingHandlerOptions
func (a *loggingGeneratingHandler) isNewResourceVersion(obj *v1beta1.Logging) bool {
	if !a.opts.UniqueApplyForResourceVersion {
		return true
	}

	// Apply once per resource version
	key := obj.Namespace + "/" + obj.Name
	previous, ok := a.seen.Load(key)
	return !ok || previous != obj.ResourceVersion
}

// storeResourceVersion keeps track of the latest resource version of an object for which Apply was executed
// Only used if UniqueApplyForResourceVersion is set in generic.GeneratingHandlerOptions
func (a *loggingGeneratingHandler) storeResourceVersion(obj *v1beta1.Logging) {
	if !a.opts.UniqueApplyForResourceVersion {
		return
	}

	key := obj.Namespace + "/" + obj.Name
	a.seen.Store(key, obj.ResourceVersion)
}
