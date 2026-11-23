package fakeclients

import (
	"context"

	mgmtv3type "github.com/harvester/harvester/pkg/generated/clientset/versioned/typed/management.cattle.io/v3"
	mgmtv3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/rancher/wrangler/v3/pkg/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type UserCache func() mgmtv3type.UserInterface

func (c UserCache) Get(name string) (*mgmtv3.User, error) {
	return c().Get(context.TODO(), name, metav1.GetOptions{})
}

func (c UserCache) List(selector labels.Selector) ([]*mgmtv3.User, error) {
	list, err := c().List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*mgmtv3.User, 0, len(list.Items))
	for _, u := range list.Items {
		obj := u
		result = append(result, &obj)
	}
	return result, err
}

func (c UserCache) AddIndexer(_ string, _ generic.Indexer[*mgmtv3.User]) {
	panic("implement me")
}

func (c UserCache) GetByIndex(_, _ string) ([]*mgmtv3.User, error) {
	panic("implement me")
}

type UserClient func() mgmtv3type.UserInterface

func (c UserClient) Update(user *mgmtv3.User) (*mgmtv3.User, error) {
	return c().Update(context.TODO(), user, metav1.UpdateOptions{})
}

func (c UserClient) Get(name string, options metav1.GetOptions) (*mgmtv3.User, error) {
	return c().Get(context.TODO(), name, options)
}

func (c UserClient) Create(user *mgmtv3.User) (*mgmtv3.User, error) {
	return c().Create(context.TODO(), user, metav1.CreateOptions{})
}

func (c UserClient) Delete(name string, options *metav1.DeleteOptions) error {
	return c().Delete(context.TODO(), name, *options)
}

func (c UserClient) List(metav1.ListOptions) (*mgmtv3.UserList, error) {
	panic("implement me")
}

func (c UserClient) Watch(metav1.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (c UserClient) Patch(_ string, _ types.PatchType, _ []byte, _ ...string) (result *mgmtv3.User, err error) {
	panic("implement me")
}

func (c UserClient) WithImpersonation(_ rest.ImpersonationConfig) (generic.NonNamespacedClientInterface[*mgmtv3.User, *mgmtv3.UserList], error) {
	panic("implement me")
}

func (c UserClient) UpdateStatus(user *mgmtv3.User) (*mgmtv3.User, error) {
	panic("implement me")
}
