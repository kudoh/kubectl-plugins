package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/util/deployment"
)

const revisionKey = "deployment.kubernetes.io/revision"

var (
	kubeconfig string
	deployName string
	namespace  string
)

func main() {

	if home := homeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(Optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.StringVar(&deployName, "d", "", "(Required) deployment resource name")
	flag.StringVar(&namespace, "n", "", "(Optional) target Namspace name(default: default)")

	flag.Parse()

	if deployName == "" {
		fmt.Println("Deployment name(-d) Required")
		os.Exit(1)
	}
	if namespace == "" {
		ns, _, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
			&clientcmd.ConfigOverrides{}).Namespace()
		if err != nil {
			panic(err.Error())
		}
		namespace = ns
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	latest, err := clientset.AppsV1().Deployments(namespace).Get(deployName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("%s not found in %s\n", deployName, namespace)
		return
	} else if err != nil {
		panic(err.Error())
	}

	_, allOldRSs, newRS, err := deployment.GetAllReplicaSets(latest, clientset.AppsV1())
	if err != nil {
		panic(err.Error())
	}
	allRSs := allOldRSs
	if newRS != nil {
		allRSs = append(allRSs, newRS)
	}
	if len(allRSs) == 0 {
		fmt.Printf("[%s] No ReplicaSet found\n", deployName)
		return
	}

	// collect revisions
	fmt.Println("Rev\tVersion\tCreatedAt\t\tImage\t\t\t\tStatus")
	for _, rs := range allRSs {
		r, err := deployment.Revision(rs)
		if err != nil {
			panic(fmt.Errorf("Cannot Get Revision for %s, %s", rs.Name, err))
		}

		var status string
		switch {
		case rs.Status.Replicas == 0:
			status = "Terminated"
		case rs.Status.Replicas == rs.Status.ReadyReplicas:
			status = "Ready"
		default:
			status = "NotReady"
		}

		fmt.Printf("%d\t%s\t%s\t%s\t%s\n", r, rs.ResourceVersion, rs.GetCreationTimestamp().Format("2006-01-02 15:04"),
			rs.Spec.Template.Spec.Containers[0].Image, status)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
