package cmd

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	v1beta12 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

type cmdFlags struct {
	ingress         *string
	method          *string
	path            *string
	https           *bool
	repeat          *bool
	skipHttpsVerify *bool
	*genericclioptions.ConfigFlags
	*genericclioptions.ResourceBuilderFlags
}

var (
	req     cmdFlags
	rootCmd *cobra.Command
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
		Short:         "call arbitrary http request",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetOut(streams.Out)
			cmd.SetErr(streams.ErrOut)
			return callHttp()
		},
	}
	req = cmdFlags{
		ingress:         rootCmd.PersistentFlags().StringP("ingress", "i", "", "Ingress name"),
		method:          rootCmd.PersistentFlags().StringP("method", "m", "GET", "HTTP Method"),
		https:           rootCmd.PersistentFlags().Bool("https", false, "use https"),
		skipHttpsVerify: rootCmd.PersistentFlags().Bool("skipVerify", false, "skip https verify"),
		path:            rootCmd.PersistentFlags().StringP("path", "p", "/", "Request Path"),
		repeat:          rootCmd.PersistentFlags().BoolP("repeat", "r", false, "repeat request"),
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

	restClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: *req.skipHttpsVerify,
			},
		},
	}

	url := fmt.Sprintf("%s://%s%s%s", protocol, rule.Host, path.Path, apipath)

	if *req.repeat {
		for {
			if err := doRequest(&url, restClient); err != nil {
				return err
			}
			time.Sleep(1 * time.Second)
		}
	} else {
		if err := doRequest(&url, restClient); err != nil {
			return err
		}
	}

	return nil
}

func getIngress(ls *v1beta12.IngressList) (*v1beta12.Ingress, error) {

	if len(ls.Items) == 1 {
		return &ls.Items[0], nil
	}

	if *req.ingress != "" {
		for _, ing := range ls.Items {
			if *req.ingress == ing.Name {
				return &ing, nil
			}
		}
		return nil, fmt.Errorf("ingress not found")
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
		if len(rule.HTTP.Paths) == 1 {
			return rule, &rule.HTTP.Paths[0], nil
		}
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
			fmt.Printf("[%d] %s%s\n", idx, r.Host, path.Path)
			idx++
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
			idx++
		}
	}

	return nil, nil, fmt.Errorf("unknown error")
}

func doRequest(url *string, restClient *http.Client) error {

	r, err := http.NewRequest(*req.method, *url, nil)
	if err != nil {
		return err
	}

	fmt.Println("requesting... ", *url)

	start := time.Now().UnixNano()
	resp, err := restClient.Do(r)
	fmt.Println("Elapsed time(ms):", (time.Now().UnixNano()-start)/1000000)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("Status:", resp.StatusCode)
	for k, v := range resp.Header {
		fmt.Println(k, ":", v)
	}
	fmt.Println(string(body))

	return nil
}
