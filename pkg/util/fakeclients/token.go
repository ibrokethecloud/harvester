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

type TokenCache func() mgmtv3type.TokenInterface

func (c TokenCache) Get(name string) (*mgmtv3.Token, error) {
	return c().Get(context.TODO(), name, metav1.GetOptions{})
}

func (c TokenCache) List(selector labels.Selector) ([]*mgmtv3.Token, error) {
	list, err := c().List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*mgmtv3.Token, 0, len(list.Items))
	for _, token := range list.Items {
		obj := token
		result = append(result, &obj)
	}
	return result, err
}

func (c TokenCache) AddIndexer(_ string, _ generic.Indexer[*mgmtv3.Token]) {
	panic("implement me")
}

func (c TokenCache) GetByIndex(_, _ string) ([]*mgmtv3.Token, error) {
	panic("implement me")
}

type TokenClient func() mgmtv3type.TokenInterface

func (c TokenClient) Update(token *mgmtv3.Token) (*mgmtv3.Token, error) {
	return c().Update(context.TODO(), token, metav1.UpdateOptions{})
}

func (c TokenClient) Get(name string, options metav1.GetOptions) (*mgmtv3.Token, error) {
	return c().Get(context.TODO(), name, options)
}

func (c TokenClient) Create(token *mgmtv3.Token) (*mgmtv3.Token, error) {
	return c().Create(context.TODO(), token, metav1.CreateOptions{})
}

func (c TokenClient) Delete(name string, options *metav1.DeleteOptions) error {
	return c().Delete(context.TODO(), name, *options)
}

func (c TokenClient) List(metav1.ListOptions) (*mgmtv3.TokenList, error) {
	panic("implement me")
}

func (c TokenClient) Watch(metav1.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (c TokenClient) Patch(_ string, _ types.PatchType, _ []byte, _ ...string) (result *mgmtv3.Token, err error) {
	panic("implement me")
}

func (c TokenClient) WithImpersonation(_ rest.ImpersonationConfig) (generic.NonNamespacedClientInterface[*mgmtv3.Token, *mgmtv3.TokenList], error) {
	panic("implement me")
}

func (c TokenClient) UpdateStatus(token *mgmtv3.Token) (*mgmtv3.Token, error) {
	panic("imeplemnt me")
}
