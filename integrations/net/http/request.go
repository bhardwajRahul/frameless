package http

import (
	"context"
	"net/http"
	"net/url"

	"github.com/adamluzsi/frameless/dataproviders"
)

type Request struct {
	srcRequest      *http.Request
	iteratorBuilder dataproviders.IteratorBuilder
}

func NewRequest(r *http.Request, payloadDecoderBuilder dataproviders.IteratorBuilder) *Request {
	return &Request{
		srcRequest:      r,
		iteratorBuilder: payloadDecoderBuilder,
	}
}

func (r *Request) Context() context.Context {
	return r.srcRequest.Context()
}

type options url.Values

func (o options) Get(key interface{}) interface{} {
	return o.GetAll(key)[0]
}

func (o options) Lookup(key interface{}) (interface{}, bool) {
	is, ok := o.LookupAll(key)

	return is[0], ok
}

func (o options) GetAll(key interface{}) []interface{} {
	vs, ok := o[key.(string)]

	if !ok {
		return nil
	}

	is := make([]interface{}, 0, len(vs))

	for _, v := range vs {
		is = append(is, v)
	}

	return is
}

func (o options) LookupAll(key interface{}) ([]interface{}, bool) {
	vs, ok := o[key.(string)]

	if !ok {
		return []interface{}{}, ok
	}

	is := make([]interface{}, 0, len(vs))

	for _, v := range vs {
		is = append(is, v)
	}

	return is, ok
}

func (r *Request) Options() dataproviders.Getter {
	return options(r.srcRequest.URL.Query())
}

func (r *Request) Data() dataproviders.Iterator {
	return r.iteratorBuilder(r.srcRequest.Body)
}

func (r *Request) Close() error {
	return r.srcRequest.Body.Close()
}