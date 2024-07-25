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

package v3

import (
	"context"
	"sync"
	"time"

	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
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

type ManagedChartHandler func(string, *v3.ManagedChart) (*v3.ManagedChart, error)

type ManagedChartController interface {
	generic.ControllerMeta
	ManagedChartClient

	OnChange(ctx context.Context, name string, sync ManagedChartHandler)
	OnRemove(ctx context.Context, name string, sync ManagedChartHandler)
	Enqueue(namespace, name string)
	EnqueueAfter(namespace, name string, duration time.Duration)

	Cache() ManagedChartCache
}

type ManagedChartClient interface {
	Create(*v3.ManagedChart) (*v3.ManagedChart, error)
	Update(*v3.ManagedChart) (*v3.ManagedChart, error)
	UpdateStatus(*v3.ManagedChart) (*v3.ManagedChart, error)
	Delete(namespace, name string, options *metav1.DeleteOptions) error
	Get(namespace, name string, options metav1.GetOptions) (*v3.ManagedChart, error)
	List(namespace string, opts metav1.ListOptions) (*v3.ManagedChartList, error)
	Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error)
	Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.ManagedChart, err error)
}

type ManagedChartCache interface {
	Get(namespace, name string) (*v3.ManagedChart, error)
	List(namespace string, selector labels.Selector) ([]*v3.ManagedChart, error)

	AddIndexer(indexName string, indexer ManagedChartIndexer)
	GetByIndex(indexName, key string) ([]*v3.ManagedChart, error)
}

type ManagedChartIndexer func(obj *v3.ManagedChart) ([]string, error)

type managedChartController struct {
	controller    controller.SharedController
	client        *client.Client
	gvk           schema.GroupVersionKind
	groupResource schema.GroupResource
}

func NewManagedChartController(gvk schema.GroupVersionKind, resource string, namespaced bool, controller controller.SharedControllerFactory) ManagedChartController {
	c := controller.ForResourceKind(gvk.GroupVersion().WithResource(resource), gvk.Kind, namespaced)
	return &managedChartController{
		controller: c,
		client:     c.Client(),
		gvk:        gvk,
		groupResource: schema.GroupResource{
			Group:    gvk.Group,
			Resource: resource,
		},
	}
}

func FromManagedChartHandlerToHandler(sync ManagedChartHandler) generic.Handler {
	return func(key string, obj runtime.Object) (ret runtime.Object, err error) {
		var v *v3.ManagedChart
		if obj == nil {
			v, err = sync(key, nil)
		} else {
			v, err = sync(key, obj.(*v3.ManagedChart))
		}
		if v == nil {
			return nil, err
		}
		return v, err
	}
}

func (c *managedChartController) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Update(obj.(*v3.ManagedChart))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func UpdateManagedChartDeepCopyOnChange(client ManagedChartClient, obj *v3.ManagedChart, handler func(obj *v3.ManagedChart) (*v3.ManagedChart, error)) (*v3.ManagedChart, error) {
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

func (c *managedChartController) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controller.RegisterHandler(ctx, name, controller.SharedControllerHandlerFunc(handler))
}

func (c *managedChartController) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), handler))
}

func (c *managedChartController) OnChange(ctx context.Context, name string, sync ManagedChartHandler) {
	c.AddGenericHandler(ctx, name, FromManagedChartHandlerToHandler(sync))
}

func (c *managedChartController) OnRemove(ctx context.Context, name string, sync ManagedChartHandler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), FromManagedChartHandlerToHandler(sync)))
}

func (c *managedChartController) Enqueue(namespace, name string) {
	c.controller.Enqueue(namespace, name)
}

func (c *managedChartController) EnqueueAfter(namespace, name string, duration time.Duration) {
	c.controller.EnqueueAfter(namespace, name, duration)
}

func (c *managedChartController) Informer() cache.SharedIndexInformer {
	return c.controller.Informer()
}

func (c *managedChartController) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *managedChartController) Cache() ManagedChartCache {
	return &managedChartCache{
		indexer:  c.Informer().GetIndexer(),
		resource: c.groupResource,
	}
}

func (c *managedChartController) Create(obj *v3.ManagedChart) (*v3.ManagedChart, error) {
	result := &v3.ManagedChart{}
	return result, c.client.Create(context.TODO(), obj.Namespace, obj, result, metav1.CreateOptions{})
}

func (c *managedChartController) Update(obj *v3.ManagedChart) (*v3.ManagedChart, error) {
	result := &v3.ManagedChart{}
	return result, c.client.Update(context.TODO(), obj.Namespace, obj, result, metav1.UpdateOptions{})
}

func (c *managedChartController) UpdateStatus(obj *v3.ManagedChart) (*v3.ManagedChart, error) {
	result := &v3.ManagedChart{}
	return result, c.client.UpdateStatus(context.TODO(), obj.Namespace, obj, result, metav1.UpdateOptions{})
}

func (c *managedChartController) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return c.client.Delete(context.TODO(), namespace, name, *options)
}

func (c *managedChartController) Get(namespace, name string, options metav1.GetOptions) (*v3.ManagedChart, error) {
	result := &v3.ManagedChart{}
	return result, c.client.Get(context.TODO(), namespace, name, result, options)
}

func (c *managedChartController) List(namespace string, opts metav1.ListOptions) (*v3.ManagedChartList, error) {
	result := &v3.ManagedChartList{}
	return result, c.client.List(context.TODO(), namespace, result, opts)
}

func (c *managedChartController) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch(context.TODO(), namespace, opts)
}

func (c *managedChartController) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*v3.ManagedChart, error) {
	result := &v3.ManagedChart{}
	return result, c.client.Patch(context.TODO(), namespace, name, pt, data, result, metav1.PatchOptions{}, subresources...)
}

type managedChartCache struct {
	indexer  cache.Indexer
	resource schema.GroupResource
}

func (c *managedChartCache) Get(namespace, name string) (*v3.ManagedChart, error) {
	obj, exists, err := c.indexer.GetByKey(namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(c.resource, name)
	}
	return obj.(*v3.ManagedChart), nil
}

func (c *managedChartCache) List(namespace string, selector labels.Selector) (ret []*v3.ManagedChart, err error) {

	err = cache.ListAllByNamespace(c.indexer, namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v3.ManagedChart))
	})

	return ret, err
}

func (c *managedChartCache) AddIndexer(indexName string, indexer ManagedChartIndexer) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(*v3.ManagedChart))
		},
	}))
}

func (c *managedChartCache) GetByIndex(indexName, key string) (result []*v3.ManagedChart, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]*v3.ManagedChart, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(*v3.ManagedChart))
	}
	return result, nil
}

// ManagedChartStatusHandler is executed for every added or modified ManagedChart. Should return the new status to be updated
type ManagedChartStatusHandler func(obj *v3.ManagedChart, status v3.ManagedChartStatus) (v3.ManagedChartStatus, error)

// ManagedChartGeneratingHandler is the top-level handler that is executed for every ManagedChart event. It extends ManagedChartStatusHandler by a returning a slice of child objects to be passed to apply.Apply
type ManagedChartGeneratingHandler func(obj *v3.ManagedChart, status v3.ManagedChartStatus) ([]runtime.Object, v3.ManagedChartStatus, error)

// RegisterManagedChartStatusHandler configures a ManagedChartController to execute a ManagedChartStatusHandler for every events observed.
// If a non-empty condition is provided, it will be updated in the status conditions for every handler execution
func RegisterManagedChartStatusHandler(ctx context.Context, controller ManagedChartController, condition condition.Cond, name string, handler ManagedChartStatusHandler) {
	statusHandler := &managedChartStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, FromManagedChartHandlerToHandler(statusHandler.sync))
}

// RegisterManagedChartGeneratingHandler configures a ManagedChartController to execute a ManagedChartGeneratingHandler for every events observed, passing the returned objects to the provided apply.Apply.
// If a non-empty condition is provided, it will be updated in the status conditions for every handler execution
func RegisterManagedChartGeneratingHandler(ctx context.Context, controller ManagedChartController, apply apply.Apply,
	condition condition.Cond, name string, handler ManagedChartGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &managedChartGeneratingHandler{
		ManagedChartGeneratingHandler: handler,
		apply:                         apply,
		name:                          name,
		gvk:                           controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterManagedChartStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type managedChartStatusHandler struct {
	client    ManagedChartClient
	condition condition.Cond
	handler   ManagedChartStatusHandler
}

// sync is executed on every resource addition or modification. Executes the configured handlers and sends the updated status to the Kubernetes API
func (a *managedChartStatusHandler) sync(key string, obj *v3.ManagedChart) (*v3.ManagedChart, error) {
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

type managedChartGeneratingHandler struct {
	ManagedChartGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
	seen  sync.Map
}

// Remove handles the observed deletion of a resource, cascade deleting every associated resource previously applied
func (a *managedChartGeneratingHandler) Remove(key string, obj *v3.ManagedChart) (*v3.ManagedChart, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v3.ManagedChart{}
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

// Handle executes the configured ManagedChartGeneratingHandler and pass the resulting objects to apply.Apply, finally returning the new status of the resource
func (a *managedChartGeneratingHandler) Handle(obj *v3.ManagedChart, status v3.ManagedChartStatus) (v3.ManagedChartStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.ManagedChartGeneratingHandler(obj, status)
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
func (a *managedChartGeneratingHandler) isNewResourceVersion(obj *v3.ManagedChart) bool {
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
func (a *managedChartGeneratingHandler) storeResourceVersion(obj *v3.ManagedChart) {
	if !a.opts.UniqueApplyForResourceVersion {
		return
	}

	key := obj.Namespace + "/" + obj.Name
	a.seen.Store(key, obj.ResourceVersion)
}
