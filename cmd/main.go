package main

import (
	"os"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/zeroFruit/operator-demo/controller"

	loggerclientset "github.com/zeroFruit/operator-demo/pkg/client/clientset/versioned"
	loggerinformersv1 "github.com/zeroFruit/operator-demo/pkg/client/informers/externalversions"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

func main() {
	// klog print to console
	klog.SetOutput(os.Stdout)

	// set up signals so we handle the first shutdown gracefully
	stopCh := make(chan struct{})
	defer close(stopCh)

	// suppose we are running operator-demo in minikube
	config := getClientConfig()
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create kubernetes client: %v", err)
	}

	loggerClient, err := loggerclientset.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create CRD logger client: %v", err)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*1000)
	loggerInformerFactory := loggerinformersv1.NewSharedInformerFactory(loggerClient, time.Second*1000)

	c := controller.New(
		kubeClient,
		loggerClient,
		kubeInformerFactory.Apps().V1().Deployments(),
		loggerInformerFactory.Example().V1().Loggers(),
	)

	// notice that there is no need to run Start methods in separate goroutine.
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine/
	kubeInformerFactory.Start(stopCh)
	loggerInformerFactory.Start(stopCh)

	if err := c.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %v", err)
	}
}

// getClientConfig retrieve the Kubernetes client from outside of the cluster
func getClientConfig() *rest.Config {
	// construct the path to resolve to `~/.kube/config`
	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		klog.Fatalf("Error get Kubernetes config: %v", err)
	}
	return config
}
