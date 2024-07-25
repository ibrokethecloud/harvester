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

	v1beta1 "github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
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

type VirtualMachineTemplateVersionHandler func(string, *v1beta1.VirtualMachineTemplateVersion) (*v1beta1.VirtualMachineTemplateVersion, error)

type VirtualMachineTemplateVersionController interface {
	generic.ControllerMeta
	VirtualMachineTemplateVersionClient

	OnChange(ctx context.Context, name string, sync VirtualMachineTemplateVersionHandler)
	OnRemove(ctx context.Context, name string, sync VirtualMachineTemplateVersionHandler)
	Enqueue(namespace, name string)
	EnqueueAfter(namespace, name string, duration time.Duration)

	Cache() VirtualMachineTemplateVersionCache
}

type VirtualMachineTemplateVersionClient interface {
	Create(*v1beta1.VirtualMachineTemplateVersion) (*v1beta1.VirtualMachineTemplateVersion, error)
	Update(*v1beta1.VirtualMachineTemplateVersion) (*v1beta1.VirtualMachineTemplateVersion, error)
	UpdateStatus(*v1beta1.VirtualMachineTemplateVersion) (*v1beta1.VirtualMachineTemplateVersion, error)
	Delete(namespace, name string, options *metav1.DeleteOptions) error
	Get(namespace, name string, options metav1.GetOptions) (*v1beta1.VirtualMachineTemplateVersion, error)
	List(namespace string, opts metav1.ListOptions) (*v1beta1.VirtualMachineTemplateVersionList, error)
	Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error)
	Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.VirtualMachineTemplateVersion, err error)
}

type VirtualMachineTemplateVersionCache interface {
	Get(namespace, name string) (*v1beta1.VirtualMachineTemplateVersion, error)
	List(namespace string, selector labels.Selector) ([]*v1beta1.VirtualMachineTemplateVersion, error)

	AddIndexer(indexName string, indexer VirtualMachineTemplateVersionIndexer)
	GetByIndex(indexName, key string) ([]*v1beta1.VirtualMachineTemplateVersion, error)
}

type VirtualMachineTemplateVersionIndexer func(obj *v1beta1.VirtualMachineTemplateVersion) ([]string, error)

type virtualMachineTemplateVersionController struct {
	controller    controller.SharedController
	client        *client.Client
	gvk           schema.GroupVersionKind
	groupResource schema.GroupResource
}

func NewVirtualMachineTemplateVersionController(gvk schema.GroupVersionKind, resource string, namespaced bool, controller controller.SharedControllerFactory) VirtualMachineTemplateVersionController {
	c := controller.ForResourceKind(gvk.GroupVersion().WithResource(resource), gvk.Kind, namespaced)
	return &virtualMachineTemplateVersionController{
		controller: c,
		client:     c.Client(),
		gvk:        gvk,
		groupResource: schema.GroupResource{
			Group:    gvk.Group,
			Resource: resource,
		},
	}
}

func FromVirtualMachineTemplateVersionHandlerToHandler(sync VirtualMachineTemplateVersionHandler) generic.Handler {
	return func(key string, obj runtime.Object) (ret runtime.Object, err error) {
		var v *v1beta1.VirtualMachineTemplateVersion
		if obj == nil {
			v, err = sync(key, nil)
		} else {
			v, err = sync(key, obj.(*v1beta1.VirtualMachineTemplateVersion))
		}
		if v == nil {
			return nil, err
		}
		return v, err
	}
}

func (c *virtualMachineTemplateVersionController) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Update(obj.(*v1beta1.VirtualMachineTemplateVersion))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func UpdateVirtualMachineTemplateVersionDeepCopyOnChange(client VirtualMachineTemplateVersionClient, obj *v1beta1.VirtualMachineTemplateVersion, handler func(obj *v1beta1.VirtualMachineTemplateVersion) (*v1beta1.VirtualMachineTemplateVersion, error)) (*v1beta1.VirtualMachineTemplateVersion, error) {
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

func (c *virtualMachineTemplateVersionController) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controller.RegisterHandler(ctx, name, controller.SharedControllerHandlerFunc(handler))
}

func (c *virtualMachineTemplateVersionController) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), handler))
}

func (c *virtualMachineTemplateVersionController) OnChange(ctx context.Context, name string, sync VirtualMachineTemplateVersionHandler) {
	c.AddGenericHandler(ctx, name, FromVirtualMachineTemplateVersionHandlerToHandler(sync))
}

func (c *virtualMachineTemplateVersionController) OnRemove(ctx context.Context, name string, sync VirtualMachineTemplateVersionHandler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), FromVirtualMachineTemplateVersionHandlerToHandler(sync)))
}

func (c *virtualMachineTemplateVersionController) Enqueue(namespace, name string) {
	c.controller.Enqueue(namespace, name)
}

func (c *virtualMachineTemplateVersionController) EnqueueAfter(namespace, name string, duration time.Duration) {
	c.controller.EnqueueAfter(namespace, name, duration)
}

func (c *virtualMachineTemplateVersionController) Informer() cache.SharedIndexInformer {
	return c.controller.Informer()
}

func (c *virtualMachineTemplateVersionController) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *virtualMachineTemplateVersionController) Cache() VirtualMachineTemplateVersionCache {
	return &virtualMachineTemplateVersionCache{
		indexer:  c.Informer().GetIndexer(),
		resource: c.groupResource,
	}
}

func (c *virtualMachineTemplateVersionController) Create(obj *v1beta1.VirtualMachineTemplateVersion) (*v1beta1.VirtualMachineTemplateVersion, error) {
	result := &v1beta1.VirtualMachineTemplateVersion{}
	return result, c.client.Create(context.TODO(), obj.Namespace, obj, result, metav1.CreateOptions{})
}

func (c *virtualMachineTemplateVersionController) Update(obj *v1beta1.VirtualMachineTemplateVersion) (*v1beta1.VirtualMachineTemplateVersion, error) {
	result := &v1beta1.VirtualMachineTemplateVersion{}
	return result, c.client.Update(context.TODO(), obj.Namespace, obj, result, metav1.UpdateOptions{})
}

func (c *virtualMachineTemplateVersionController) UpdateStatus(obj *v1beta1.VirtualMachineTemplateVersion) (*v1beta1.VirtualMachineTemplateVersion, error) {
	result := &v1beta1.VirtualMachineTemplateVersion{}
	return result, c.client.UpdateStatus(context.TODO(), obj.Namespace, obj, result, metav1.UpdateOptions{})
}

func (c *virtualMachineTemplateVersionController) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return c.client.Delete(context.TODO(), namespace, name, *options)
}

func (c *virtualMachineTemplateVersionController) Get(namespace, name string, options metav1.GetOptions) (*v1beta1.VirtualMachineTemplateVersion, error) {
	result := &v1beta1.VirtualMachineTemplateVersion{}
	return result, c.client.Get(context.TODO(), namespace, name, result, options)
}

func (c *virtualMachineTemplateVersionController) List(namespace string, opts metav1.ListOptions) (*v1beta1.VirtualMachineTemplateVersionList, error) {
	result := &v1beta1.VirtualMachineTemplateVersionList{}
	return result, c.client.List(context.TODO(), namespace, result, opts)
}

func (c *virtualMachineTemplateVersionController) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch(context.TODO(), namespace, opts)
}

func (c *virtualMachineTemplateVersionController) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*v1beta1.VirtualMachineTemplateVersion, error) {
	result := &v1beta1.VirtualMachineTemplateVersion{}
	return result, c.client.Patch(context.TODO(), namespace, name, pt, data, result, metav1.PatchOptions{}, subresources...)
}

type virtualMachineTemplateVersionCache struct {
	indexer  cache.Indexer
	resource schema.GroupResource
}

func (c *virtualMachineTemplateVersionCache) Get(namespace, name string) (*v1beta1.VirtualMachineTemplateVersion, error) {
	obj, exists, err := c.indexer.GetByKey(namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(c.resource, name)
	}
	return obj.(*v1beta1.VirtualMachineTemplateVersion), nil
}

func (c *virtualMachineTemplateVersionCache) List(namespace string, selector labels.Selector) (ret []*v1beta1.VirtualMachineTemplateVersion, err error) {

	err = cache.ListAllByNamespace(c.indexer, namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1beta1.VirtualMachineTemplateVersion))
	})

	return ret, err
}

func (c *virtualMachineTemplateVersionCache) AddIndexer(indexName string, indexer VirtualMachineTemplateVersionIndexer) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(*v1beta1.VirtualMachineTemplateVersion))
		},
	}))
}

func (c *virtualMachineTemplateVersionCache) GetByIndex(indexName, key string) (result []*v1beta1.VirtualMachineTemplateVersion, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]*v1beta1.VirtualMachineTemplateVersion, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(*v1beta1.VirtualMachineTemplateVersion))
	}
	return result, nil
}

// VirtualMachineTemplateVersionStatusHandler is executed for every added or modified VirtualMachineTemplateVersion. Should return the new status to be updated
type VirtualMachineTemplateVersionStatusHandler func(obj *v1beta1.VirtualMachineTemplateVersion, status v1beta1.VirtualMachineTemplateVersionStatus) (v1beta1.VirtualMachineTemplateVersionStatus, error)

// VirtualMachineTemplateVersionGeneratingHandler is the top-level handler that is executed for every VirtualMachineTemplateVersion event. It extends VirtualMachineTemplateVersionStatusHandler by a returning a slice of child objects to be passed to apply.Apply
type VirtualMachineTemplateVersionGeneratingHandler func(obj *v1beta1.VirtualMachineTemplateVersion, status v1beta1.VirtualMachineTemplateVersionStatus) ([]runtime.Object, v1beta1.VirtualMachineTemplateVersionStatus, error)

// RegisterVirtualMachineTemplateVersionStatusHandler configures a VirtualMachineTemplateVersionController to execute a VirtualMachineTemplateVersionStatusHandler for every events observed.
// If a non-empty condition is provided, it will be updated in the status conditions for every handler execution
func RegisterVirtualMachineTemplateVersionStatusHandler(ctx context.Context, controller VirtualMachineTemplateVersionController, condition condition.Cond, name string, handler VirtualMachineTemplateVersionStatusHandler) {
	statusHandler := &virtualMachineTemplateVersionStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, FromVirtualMachineTemplateVersionHandlerToHandler(statusHandler.sync))
}

// RegisterVirtualMachineTemplateVersionGeneratingHandler configures a VirtualMachineTemplateVersionController to execute a VirtualMachineTemplateVersionGeneratingHandler for every events observed, passing the returned objects to the provided apply.Apply.
// If a non-empty condition is provided, it will be updated in the status conditions for every handler execution
func RegisterVirtualMachineTemplateVersionGeneratingHandler(ctx context.Context, controller VirtualMachineTemplateVersionController, apply apply.Apply,
	condition condition.Cond, name string, handler VirtualMachineTemplateVersionGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &virtualMachineTemplateVersionGeneratingHandler{
		VirtualMachineTemplateVersionGeneratingHandler: handler,
		apply: apply,
		name:  name,
		gvk:   controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterVirtualMachineTemplateVersionStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type virtualMachineTemplateVersionStatusHandler struct {
	client    VirtualMachineTemplateVersionClient
	condition condition.Cond
	handler   VirtualMachineTemplateVersionStatusHandler
}

// sync is executed on every resource addition or modification. Executes the configured handlers and sends the updated status to the Kubernetes API
func (a *virtualMachineTemplateVersionStatusHandler) sync(key string, obj *v1beta1.VirtualMachineTemplateVersion) (*v1beta1.VirtualMachineTemplateVersion, error) {
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

type virtualMachineTemplateVersionGeneratingHandler struct {
	VirtualMachineTemplateVersionGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
	seen  sync.Map
}

// Remove handles the observed deletion of a resource, cascade deleting every associated resource previously applied
func (a *virtualMachineTemplateVersionGeneratingHandler) Remove(key string, obj *v1beta1.VirtualMachineTemplateVersion) (*v1beta1.VirtualMachineTemplateVersion, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v1beta1.VirtualMachineTemplateVersion{}
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

// Handle executes the configured VirtualMachineTemplateVersionGeneratingHandler and pass the resulting objects to apply.Apply, finally returning the new status of the resource
func (a *virtualMachineTemplateVersionGeneratingHandler) Handle(obj *v1beta1.VirtualMachineTemplateVersion, status v1beta1.VirtualMachineTemplateVersionStatus) (v1beta1.VirtualMachineTemplateVersionStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.VirtualMachineTemplateVersionGeneratingHandler(obj, status)
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
func (a *virtualMachineTemplateVersionGeneratingHandler) isNewResourceVersion(obj *v1beta1.VirtualMachineTemplateVersion) bool {
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
func (a *virtualMachineTemplateVersionGeneratingHandler) storeResourceVersion(obj *v1beta1.VirtualMachineTemplateVersion) {
	if !a.opts.UniqueApplyForResourceVersion {
		return
	}

	key := obj.Namespace + "/" + obj.Name
	a.seen.Store(key, obj.ResourceVersion)
}
