package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"http/cmd/check"
	"io/ioutil"
	v1beta12 "k8s.io/api/networking/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

type cmdFlags struct {
	ingress         *string
	https           *bool
	repeat          *bool
	skipHttpsVerify *bool
	*genericclioptions.ConfigFlags
	//	*genericclioptions.ResourceBuilderFlags
}

var (
	req         cmdFlags
	rootCmd     *cobra.Command
	httpMethods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
)

func init() {

	flags := pflag.NewFlagSet("kubectl-http", pflag.ExitOnError)

	pflag.CommandLine = flags

	streams := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	rootCmd = &cobra.Command{
		Use:           "kubectl http",
		Short:         "check arbitrary http request",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetOut(streams.Out)
			cmd.SetErr(streams.ErrOut)
			return test()
		},
	}
	req = cmdFlags{
		ingress:         rootCmd.PersistentFlags().StringP("ingress", "i", "", "Ingress name"),
		https:           rootCmd.PersistentFlags().Bool("https", false, "use https"),
		skipHttpsVerify: rootCmd.PersistentFlags().BoolP("skip-ingress-https-verify", "k", false, "skip https verify"),
		repeat:          rootCmd.PersistentFlags().BoolP("repeat", "r", false, "if true, calling request repeatedly"),
	}
	req.ConfigFlags = genericclioptions.NewConfigFlags(false)
	//req.ResourceBuilderFlags = genericclioptions.NewResourceBuilderFlags()

	flags.AddFlagSet(rootCmd.PersistentFlags())
	req.ConfigFlags.AddFlags(flags)
	//req.ResourceBuilderFlags.AddFlags(flags)
}

func Execute() error {
	return rootCmd.Execute()
}

func test() error {

	config, err := req.ToRESTConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	ns, _, err := req.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	ingresses, err := clientset.NetworkingV1beta1().Ingresses(ns).List(v1.ListOptions{})
	if err != nil {
		return err
	} else if len(ingresses.Items) == 0 {
		fmt.Printf("ingress not found in %s\n", ns)
		return nil
	}

	ing, err := getIngress(ingresses)
	if err != nil {
		return err
	}

	rule, err := getHost(ing)
	if err != nil {
		return err
	}

	method, err := getMethod()
	if err != nil {
		return err
	}

	apipath, err := readStringFromStdIn("path")
	if err != nil {
		return err
	}

	protocol := "http"
	if *req.https {
		protocol = "https"
	}

	var body string
	var contentType string
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		body, err = readStringFromStdIn("body")
		if err != nil {
			return err
		}
		b := regexp.MustCompile(`.+(\.json|\.xml|\.txt)$`).FindStringSubmatch(body)
		if b != nil {
			body, err = readFile(body)
			if err != nil {
				return err
			}
			switch b[1] {
			case ".json":
				contentType = "application/json"
			case ".xml":
				contentType = "application/xml"
			case ".txt":
				contentType = "text/plain"
			default:
				contentType = "application/json"
			}
		}
	}

	checker := &check.HttpChecker{
		Method: method,
		Url: &url.URL{
			Host:   rule.Host,
			Path:   apipath,
			Scheme: protocol,
		},
		Body:            strings.NewReader(body),
		SkipHttpsVerify: *req.skipHttpsVerify,
		Repeat:          *req.repeat,
		ContentType:     contentType,
	}

	if err = checker.Execute(); err != nil {
		return err
	}

	return nil
}

func getIngress(ls *v1beta12.IngressList) (*v1beta12.Ingress, error) {

	if *req.ingress != "" {
		for _, ing := range ls.Items {
			if *req.ingress == ing.Name {
				return &ing, nil
			}
		}
		return nil, fmt.Errorf("ingress not found")
	}

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

func getHost(ing *v1beta12.Ingress) (*v1beta12.IngressRule, error) {

	if len(ing.Spec.Rules) == 1 {
		rule := &ing.Spec.Rules[0]
		return rule, nil
	}

	for i, r := range ing.Spec.Rules {
		var backend [][]string
		for _, p := range r.HTTP.Paths {
			backend = append(backend, []string{p.Path, p.Backend.ServiceName, p.Backend.ServicePort.String()})
		}
		fmt.Printf("[%d] %s %v\n", i+1, r.Host, backend)
	}
	num, err := readNumFromStdIn(len(ing.Spec.Rules) + 1)
	if err != nil {
		return nil, err
	}
	return &ing.Spec.Rules[num-1], nil
}

func getMethod() (string, error) {
	for i, m := range httpMethods {
		fmt.Printf("[%d] %s\n", i+1, m)
	}
	num, err := readNumFromStdIn(len(httpMethods) + 1)
	if err != nil {
		return "", err
	}
	return httpMethods[num-1], nil
}

func readFile(path string) (string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
