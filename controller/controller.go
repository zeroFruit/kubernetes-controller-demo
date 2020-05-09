package controller

import (
	"fmt"
	"strconv"
	"time"

	v1 "github.com/zeroFruit/operator-demo/pkg/apis/logger/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/klog"

	"k8s.io/client-go/kubernetes/scheme"

	"k8s.io/client-go/tools/record"

	loggerv1 "github.com/zeroFruit/operator-demo/pkg/apis/logger/v1"
	loggerclientset "github.com/zeroFruit/operator-demo/pkg/client/clientset/versioned"
	loggerschemev1 "github.com/zeroFruit/operator-demo/pkg/client/clientset/versioned/scheme"
	loggerinformersv1 "github.com/zeroFruit/operator-demo/pkg/client/informers/externalversions/logger/v1"
	loggerlisterv1 "github.com/zeroFruit/operator-demo/pkg/client/listers/logger/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
)

const controllerAgentName = "logger-controller"
const loggerKind = "Logger"
const loggerDockerImage = "user_activity_logger:latest"

type Controller struct {
	kubeclient        kubernetes.Interface
	loggerclient      loggerclientset.Interface
	deploymentsLister appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced
	loggerLister      loggerlisterv1.LoggerLister
	loggerSynced      cache.InformerSynced

	// queue is a rate limited queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens.
	// This means we can ensure we only process a fixed amount of resources
	// at a time, and makes it easy to ensure we are never processing the
	// same item simultaneously in two different controllers.
	queue workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func New(
	kubeclient kubernetes.Interface,
	loggerclient loggerclientset.Interface,
	deploymentInformer appsinformers.DeploymentInformer,
	loggerInformer loggerinformersv1.LoggerInformer) *Controller {

	// create event broadcaster
	// add Logger types to the default Kubernetes Scheme so events
	// can be logged for Logger types
	utilruntime.Must(loggerschemev1.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eb := record.NewBroadcaster()
	eb.StartLogging(klog.Infof)
	eb.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: kubeclient.CoreV1().Events("default"),
	})
	recorder := eb.NewRecorder(scheme.Scheme, corev1.EventSource{
		Component: controllerAgentName,
	})

	c := &Controller{
		kubeclient:        kubeclient,
		loggerclient:      loggerclient,
		deploymentsLister: deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		loggerLister:      loggerInformer.Lister(),
		loggerSynced:      loggerInformer.Informer().HasSynced,
		queue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Workers"),
		recorder:          recorder,
	}

	klog.Info("Setting up event handlers")
	// Setup an event handler for when Logger resources change
	loggerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueWorker,
		UpdateFunc: func(old, new interface{}) {
			c.enqueueWorker(new)
		},
		//DeleteFunc: c.enqueueWorker,
	})

	// Setup an event handler for when Deployment resources change. This handler
	// will lookup the owner of the given Deployment, and if it is owned by
	// a Logger resource will enqueue that Logger resource for that Logger
	// resource for processing. This way, we don't need to implement custom logic
	// for handling Deployment resources.
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDeployment := new.(*appsv1.Deployment)
			oldDeployment := old.(*appsv1.Deployment)
			if newDeployment.ResourceVersion == oldDeployment.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different ResourceVersions.
				return
			}
			c.handleObject(new)
		},
		DeleteFunc: c.handleObject,
	})

	return c
}

// Run will setup the event handlers for types we are interested in, as well as syncing
// informer caches and starting loggers. It will block until stopCh is closed, at which point
// it will shutdown the queue and wait for loggers to finish processing their current work items.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting Controller")

	// Wait for all caches to be synced, before processing items from the queue is started
	klog.Info("Waiting for informer caches to sync")
	if !cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.loggerSynced) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launching additional goroutines would parallelize workers consuming from the queue
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNext function in order to read and process a message on the queue.
func (c *Controller) runWorker() {
	for c.processNext() {
	}
}

// processNext will read a single work item off the queue and
// attempt to process it, by calling the syncHandler
func (c *Controller) processNext() bool {
	// Wait until there is a new item in the working queue
	item, quit := c.queue.Get()
	if quit {
		return false
	}
	//// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	//// This allows safe parallel processing because two pods with the same key are never processed in parallel.
	//defer c.queue.Done(key)

	// Wrap this block in a block so we can defer c.queue.Done.
	err := func(item interface{}) error {
		// We call Done here so the queue knows we have finished processing
		// this item. We also must remember to call Forget if we do not want this
		// work item being re-queued. For example, we do not call Forget if a transient
		// error occurs, instead the item is put back on the queue and attempted again after
		// a back-off period.
		defer c.queue.Done(item)
		var key string
		var ok bool

		// We expect strings to come off the queue. These are of the form namespace/name
		// We do this as the delayed nature of the queue means the items in the
		// informer cache may actually be more up to date that when the item was
		// initially put onto the queue.
		if key, ok = item.(string); !ok {
			// As the item in the queue is actually invalid, we call Forget
			// here else we'd go into a loop of attempting to process a work item that is invalid.
			c.queue.Forget(item)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", item))
			return nil
		}

		// Run the syncHandler, passing it the namespace/name string of the
		// Logger resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the queue to handle any transient errors.
			c.queue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.queue.Forget(item)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(item)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Logger resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Logger resource with this namespace/name
	logger, err := c.loggerLister.Loggers(namespace).Get(name)
	if err != nil {
		// The Logger resource may no longer exist, in which case we stop processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("logger '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	loggerName := logger.Spec.Name
	if loggerName == "" {
		// We choose to absort the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be enqueued again.
		utilruntime.HandleError(fmt.Errorf("%s: deployment name must be specified", key))
		return nil
	}

	// Get the deployment with the name specified in Logger.Spec
	deployment, err := c.deploymentsLister.Deployments(logger.Namespace).Get(name)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		deployment, err = c.kubeclient.AppsV1().Deployments(logger.Namespace).Create(newDeployment(logger))
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a temporary
	// network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the Deployment is not controlled by this Logger resource, we should log
	// a warning to the event recorder and return error message.
	if !metav1.IsControlledBy(deployment, logger) {
		msg := fmt.Sprintf("Resource %q already exists and is not managed by Logger", deployment.Name)
		c.recorder.Event(logger, corev1.EventTypeWarning, "ErrResourceExists", msg)
		return fmt.Errorf(msg)
	}

	// If this number of the replicas on the Foo resource is specified, and the
	// number does not equal the current desired replicas on the Deployment, we
	// should update the Deployment resource.
	if logger.Spec.Replicas != nil && *logger.Spec.Replicas != *deployment.Spec.Replicas {
		klog.Infof("Logger %s replicas: %d, deployment replicas: %d", name, *logger.Spec.Replicas, *deployment.Spec.Replicas)
		deployment, err = c.kubeclient.AppsV1().Deployments(logger.Namespace).Update(newDeployment(logger))
	}

	// Finally, we update the status block of thee Logger resource to reflect the
	// current state of the world
	if err = c.updateLoggerStatus(logger, deployment); err != nil {
		return err
	}

	c.recorder.Event(logger, corev1.EventTypeNormal, "Synced", fmt.Sprintf("Logger '%s' successfully synced", key))
	return nil
}

func (c *Controller) updateLoggerStatus(logger *v1.Logger, deployment *appsv1.Deployment) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	loggerCopy := logger.DeepCopy()
	loggerCopy.Status.Value = v1.Available

	// If the CustomResourceSubresources feature gate is not enabled.
	// we must use Update instead of UpdateStatus to update the Status bloock of the Logger resource.
	// UpdateStatus will not allow changes to the Spec of the resource.
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.loggerclient.ExampleV1().Loggers(logger.Namespace).Update(loggerCopy)
	return err
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the Logger resource that owns it. It does this by looking at the objects
// metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Logger resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}

	klog.Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a Logger, we should not do anything more with it.
		if ownerRef.Kind != loggerKind {
			return
		}
		logger, err := c.loggerLister.Loggers(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.Infof("ignoring orphaned object '%s' of foo '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}
		c.enqueueWorker(logger)
		return
	}
}

// enqueueWorker takes a Logger resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should not be
// passed resources of any type other than Logger.
func (c *Controller) enqueueWorker(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.queue.Add(key)
}

// newDeployment creates a new Deployment for a Logger resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the Logger resource that 'owns' it
func newDeployment(logger *v1.Logger) *appsv1.Deployment {
	labels := map[string]string{
		"app":  "user_activity_logger",
		"name": logger.Spec.Name,
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      logger.Spec.Name,
			Namespace: logger.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(logger, loggerv1.SchemeGroupVersion.WithKind("Logger")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: logger.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            logger.Spec.Name,
							ImagePullPolicy: corev1.PullNever,
							Image:           loggerDockerImage,
							Args: []string{
								logger.Spec.Name,
								strconv.Itoa(logger.Spec.TimeInterval),
							},
						},
					},
				},
			},
		},
	}
}
