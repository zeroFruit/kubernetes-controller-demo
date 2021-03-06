/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"time"

	v1 "github.com/zeroFruit/operator-demo/pkg/apis/logger/v1"
	scheme "github.com/zeroFruit/operator-demo/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// LoggersGetter has a method to return a LoggerInterface.
// A group's client should implement this interface.
type LoggersGetter interface {
	Loggers(namespace string) LoggerInterface
}

// LoggerInterface has methods to work with Logger resources.
type LoggerInterface interface {
	Create(*v1.Logger) (*v1.Logger, error)
	Update(*v1.Logger) (*v1.Logger, error)
	UpdateStatus(*v1.Logger) (*v1.Logger, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.Logger, error)
	List(opts metav1.ListOptions) (*v1.LoggerList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Logger, err error)
	LoggerExpansion
}

// loggers implements LoggerInterface
type loggers struct {
	client rest.Interface
	ns     string
}

// newLoggers returns a Loggers
func newLoggers(c *ExampleV1Client, namespace string) *loggers {
	return &loggers{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the logger, and returns the corresponding logger object, and an error if there is any.
func (c *loggers) Get(name string, options metav1.GetOptions) (result *v1.Logger, err error) {
	result = &v1.Logger{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("loggers").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Loggers that match those selectors.
func (c *loggers) List(opts metav1.ListOptions) (result *v1.LoggerList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.LoggerList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("loggers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested loggers.
func (c *loggers) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("loggers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a logger and creates it.  Returns the server's representation of the logger, and an error, if there is any.
func (c *loggers) Create(logger *v1.Logger) (result *v1.Logger, err error) {
	result = &v1.Logger{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("loggers").
		Body(logger).
		Do().
		Into(result)
	return
}

// Update takes the representation of a logger and updates it. Returns the server's representation of the logger, and an error, if there is any.
func (c *loggers) Update(logger *v1.Logger) (result *v1.Logger, err error) {
	result = &v1.Logger{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("loggers").
		Name(logger.Name).
		Body(logger).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *loggers) UpdateStatus(logger *v1.Logger) (result *v1.Logger, err error) {
	result = &v1.Logger{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("loggers").
		Name(logger.Name).
		SubResource("status").
		Body(logger).
		Do().
		Into(result)
	return
}

// Delete takes name of the logger and deletes it. Returns an error if one occurs.
func (c *loggers) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("loggers").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *loggers) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("loggers").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched logger.
func (c *loggers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Logger, err error) {
	result = &v1.Logger{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("loggers").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
