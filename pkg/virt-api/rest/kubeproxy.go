package rest

import (
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/go-kit/kit/endpoint"
	"golang.org/x/net/context"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/runtime"
	"k8s.io/client-go/1.5/rest"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/middleware"
	"kubevirt.io/kubevirt/pkg/rest/endpoints"
	"reflect"
)

type ResponseHandlerFunc func(rest.Result) (runtime.Object, error)

func AddGenericResourceProxy(ws *restful.WebService, ctx context.Context, gvr unversioned.GroupVersionResource, ptr runtime.Object, response ResponseHandlerFunc) error {
	cli, err := kubecli.GetRESTClient()
	if err != nil {
		return err
	}
	example := reflect.ValueOf(ptr).Elem().Interface()
	delete := endpoints.NewHandlerBuilder().Delete().Endpoint(NewGenericDeleteEndpoint(cli, gvr, response)).Build(ctx)
	put := endpoints.NewHandlerBuilder().Put(ptr).Endpoint(NewGenericPutEndpoint(cli, gvr, response)).Build(ctx)
	post := endpoints.NewHandlerBuilder().Post(ptr).Endpoint(NewGenericPostEndpoint(cli, gvr, response)).Build(ctx)
	get := endpoints.NewHandlerBuilder().Get().Endpoint(NewGenericGetEndpoint(cli, gvr, response)).Build(ctx)

	ws.Route(ws.POST(fmt.Sprintf("apis/%s/%s/namespaces/{namespace}/%s", gvr.Group, gvr.Version, gvr.Resource)).
		To(endpoints.MakeGoRestfulWrapper(post)).Reads(example).Writes(example))

	ws.Route(ws.PUT(fmt.Sprintf("apis/%s/%s/namespaces/{namespace}/%s/{name}", gvr.Group, gvr.Version, gvr.Resource)).
		To(endpoints.MakeGoRestfulWrapper(put)).Reads(example).Writes(example).Doc("test2"))

	ws.Route(ws.DELETE(fmt.Sprintf("apis/%s/%s/namespaces/{namespace}/%s/{name}", gvr.Group, gvr.Version, gvr.Resource)).
		To(endpoints.MakeGoRestfulWrapper(delete)).Writes(unversioned.Status{}).Doc("test3"))

	ws.Route(ws.GET(fmt.Sprintf("apis/%s/%s/namespaces/{namespace}/%s/{name}", gvr.Group, gvr.Version, gvr.Resource)).
		To(endpoints.MakeGoRestfulWrapper(get)).Writes(example).Doc("test4"))
	return nil
}

func NewGenericDeleteEndpoint(cli *rest.RESTClient, gvr unversioned.GroupVersionResource, response ResponseHandlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		metadata := payload.(*endpoints.Metadata)
		result := cli.Delete().Namespace(metadata.Namespace).Resource(gvr.Resource).Name(metadata.Name).Do()
		return response(result)
	}
}

func NewGenericPutEndpoint(cli *rest.RESTClient, gvr unversioned.GroupVersionResource, response ResponseHandlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		obj := payload.(*endpoints.PutObject)
		result := cli.Put().Namespace(obj.Metadata.Namespace).Resource(gvr.Resource).Name(obj.Metadata.Name).Body(obj.Payload).Do()
		return response(result)
	}
}

func NewGenericPostEndpoint(cli *rest.RESTClient, gvr unversioned.GroupVersionResource, response ResponseHandlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		obj := payload.(*endpoints.PutObject)
		result := cli.Post().Namespace(obj.Metadata.Namespace).Resource(gvr.Resource).Body(obj.Payload).Do()
		return response(result)
	}
}

func NewGenericGetEndpoint(cli *rest.RESTClient, gvr unversioned.GroupVersionResource, response ResponseHandlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		metadata := payload.(*endpoints.Metadata)
		result := cli.Get().Namespace(metadata.Namespace).Resource(gvr.Resource).Name(metadata.Name).Do()
		return response(result)
	}
}

//FIXME this is basically one big workaround because version and kind are not filled by the restclient
func NewResponseHandler(gvk unversioned.GroupVersionKind, ptr runtime.Object) ResponseHandlerFunc {
	return func(result rest.Result) (runtime.Object, error) {
		if result.Error() != nil {
			return nil, middleware.NewInternalServerError(result.Error())
		}
		obj, err := result.Get()
		if reflect.TypeOf(obj).Elem() == reflect.TypeOf(ptr).Elem() {
			obj.(runtime.Object).GetObjectKind().SetGroupVersionKind(gvk)
		}
		if err != nil {
			return nil, middleware.NewInternalServerError(result.Error())
		}
		return obj, nil

	}
}
