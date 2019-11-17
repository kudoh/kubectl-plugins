package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io/ioutil"
	v1beta12 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"os"
	"strings"
)

type cmdFlags struct {
	ingress *string
	method  *string
	path    *string
	https   *bool
	*genericclioptions.ConfigFlags
	*genericclioptions.ResourceBuilderFlags
}

var (
	req     cmdFlags
	streams = genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	rootCmd = &cobra.Command{
		Use:   "kubectl http",
		Short: "call arbitrary http request",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetOut(streams.Out)
			cmd.SetErr(streams.ErrOut)
			return callHttp()
		},
	}
)

func init() {

	flags := pflag.NewFlagSet("kubectl-http", pflag.ExitOnError)

	pflag.CommandLine = flags

	req = cmdFlags{
		ingress: rootCmd.PersistentFlags().StringP("ingress", "i", "", "Ingress name"),
		method:  rootCmd.PersistentFlags().StringP("method", "m", "GET", "HTTP Method"),
		https:   rootCmd.PersistentFlags().Bool("https", false, "use https"),
		path:    rootCmd.PersistentFlags().StringP("path", "p", "/", "Request Path"),
	}
	req.ConfigFlags = genericclioptions.NewConfigFlags(false)
	req.ResourceBuilderFlags = genericclioptions.NewResourceBuilderFlags()

	flags.AddFlagSet(rootCmd.PersistentFlags())
	req.ConfigFlags.AddFlags(flags)
	req.ResourceBuilderFlags.AddFlags(flags)
}

func Execute() error {
	return rootCmd.Execute()
}

func callHttp() error {

	config, err := req.ToRESTConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	//	fmt.Println("Target Namespace: ", *req.Namespace)

	ingresses, err := clientset.NetworkingV1beta1().Ingresses(*req.Namespace).List(v1.ListOptions{})

	if errors.IsNotFound(err) {
		fmt.Printf("ingress not found in %s\n", *req.Namespace)
		return nil
	} else if err != nil {
		return err
	}

	ing, err := getIngress(ingresses)
	if err != nil {
		return err
	}

	rule, path, err := getHostAndRule(ing)
	if err != nil {
		return err
	}

	apipath, err := readStringFromStdIn("path")
	if err != nil {
		return err
	}

	if strings.HasPrefix(apipath, "/") && strings.HasSuffix(path.Path, "/") {
		apipath = strings.TrimPrefix(apipath, "/")
	} else if !strings.HasPrefix(apipath, "/") && !strings.HasSuffix(path.Path, "/") {
		apipath = "/" + apipath
	}
	protocol := "http"
	if *req.https {
		protocol = "https"
	}
	url := fmt.Sprintf("%s://%s%s%s", protocol, rule.Host, path.Path, apipath)
	fmt.Println("requesting... ", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("Status: ", resp.StatusCode)

	fmt.Println(string(body))

	return nil
}

func getIngress(ls *v1beta12.IngressList) (*v1beta12.Ingress, error) {

	if len(ls.Items) == 1 {
		return &ls.Items[0], nil
	}

	for i, ing := range ls.Items {
		fmt.Printf("[%d] %s\n", i+1, ing.Name)
	}
	num, err := readNumFromStdIn(len(ls.Items) + 1)
	if err != nil {
		return nil, err
	}
	return &ls.Items[num-1], nil
}

func getHostAndRule(ing *v1beta12.Ingress) (*v1beta12.IngressRule, *v1beta12.HTTPIngressPath, error) {

	if len(ing.Spec.Rules) == 1 {
		rule := &ing.Spec.Rules[0]
		for i, path := range rule.HTTP.Paths {
			fmt.Printf("[%d] %s%s\n", i+1, rule.Host, path.Path)
		}
		num, err := readNumFromStdIn(len(rule.HTTP.Paths) + 1)
		if err != nil {
			return nil, nil, err
		}
		path := &rule.HTTP.Paths[num-1]
		return rule, path, nil
	}

	idx := 1
	for _, r := range ing.Spec.Rules {
		for _, path := range r.HTTP.Paths {
			idx++
			fmt.Printf("[%d] %s%s\n", idx, r.Host, path.Path)
		}
	}
	num, err := readNumFromStdIn(idx)
	if err != nil {
		return nil, nil, err
	}

	idx = 1
	for _, r := range ing.Spec.Rules {
		for _, p := range r.HTTP.Paths {
			if idx == num {
				return &r, &p, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("unknown error")
}
