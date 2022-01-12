package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/ministryofjustice/cloud-platform-environments/pkg/authenticate"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	bucket          = flag.String("bucket", os.Getenv("KUBECONFIG_S3_BUCKET"), "AWS S3 bucket for kubeconfig")
	ctxLive         = flag.String("contextLive", "live.cloud-platform.service.justice.gov.uk", "Kubernetes context specified in kubeconfig")
	ctxManager      = flag.String("contextManager", "manager.cloud-platform.service.justice.gov.uk", "Kubernetes context specified in kubeconfig")
	ctxLive_1       = flag.String("contextLive_1", "live-1.cloud-platform.service.justice.gov.uk", "Kubernetes context specified in kubeconfig")
	hoodawApiKey    = flag.String("hoodawAPIKey", os.Getenv("HOODAW_API_KEY"), "API key to post data to the 'How out of date are we' API")
	hoodawEndpoint  = flag.String("hoodawEndpoint", "/ingress_weighting", "Endpoint to send the data to")
	hoodawHost      = flag.String("hoodawHost", os.Getenv("HOODAW_HOST"), "Hostname of the 'How out of date are we' API")
	kubeconfig      = flag.String("kubeconfig", "kubeconfig", "Name of kubeconfig file in S3 bucket")
	region          = flag.String("region", os.Getenv("AWS_REGION"), "AWS Region")
	KubeConfigPath  = flag.String("KubeConfigPath", "/tmp/config", "kubectl config path")

	endPoint = *hoodawHost + *hoodawEndpoint
)

type helmNamespace struct {
	Namespace string 
}

type helmRelease struct {
	Name string `json:"name"`
	Namespace string `json:"namespace"`
	InstalledVersion string `json:"installed_version"`
	LatestVersion string `json:"latest_version"`
	Chart string `json:"chart"`
}

type resourceMap map[string]interface{}

func main() {

	// Gain access to a Kubernetes cluster using a config file stored in an S3 bucket.

	err := authenticate.KubeConfigFromS3Bucket(*bucket, *kubeconfig, *region)
	if err != nil {
		log.Fatalln("error in getting config")
	}

	// kube context switch to Manger and output the results of `helm whatup`

	_, err = switchContext(*ctxManager, *KubeConfigPath)
	if err != nil {
		log.Fatalln("error switching context")
	}

	listReleaseManager, err := getAllHelmReleases()
	if err != nil {
		log.Fatalln("error in getting helm releases")
	}

	fmt.Println(listReleaseManager)

	var clusters map[string]interface{}
    clusters = {"name:", "live-1"}; {"apps:", helm_releases}

    jsonToPost, err := BuildJsonMap(clusters)
    if err != nil {
     log.Fatalln(err.Error())
    }

    // Post json to hoowdaw api
    err = hoodaw.PostToApi(jsonToPost, hoodawApiKey, &endPoint)
    if err != nil {
     log.Fatalln(err.Error())
    }
}

func getAllHelmReleases() ([]helmRelease, error){

	// Get al helm releases in namespaces
	var releases []helmRelease
	namespaces, err := namespacesWithHelmReleases()
	if err != nil {
		return nil, err
	}
	
	for _, ns := range namespaces{
		release, err := helmReleasesInNamespace(ns)
		if err != nil {
			log.Fatalln(err.Error())
		}
		releases = append(releases, release...)
	}
	return releases, nil
}

func namespacesWithHelmReleases() ([]string, error){
	cmd := exec.Command("helm", "list", "--all-namespaces", "-o", "json")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	helmListJson := out.String()

	var namespaces []helmNamespace
	json.Unmarshal([]byte(helmListJson), &namespaces)
	
	var nsList []string
	for ns := range namespaces {
		nsList = append(nsList, namespaces[ns].Namespace)
		
	}
	return deduplicateList(nsList), nil
}

func deduplicateList(s []string) (list []string) {
	keys := make(map[string]bool)

	for _, entry := range s {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}

	return
}

func helmReleasesInNamespace(namespace string) ([]helmRelease, error){
	cmd := exec.Command("helm", "whatup", "--namespace", namespace, "-o", "json")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		fmt.Println("Error: ", err)
		return nil, err
	}
	helmWhatupJson := out.String()

	fmt.Println(helmWhatupJson)

	var releases []helmRelease
	json.Unmarshal([]byte(helmWhatupJson), &releases)

	return releases, nil

}

// BuildJsonMap takes a slice of maps and return a json encoded map
func BuildJsonMap(clusters []helmRelease) ([]byte, error) {
	// To handle generics in the data type, we need to create a new map,
	// add the first key string:string and then the second key/value string:map[string]string.
	// As per the requirements of the HOODAW API.
	jsonMap := resourceMap{
		"updated_at":        time.Now().Format("2006-01-2 15:4:5 UTC"),
		"clusters": clusters,
	}

	jsonStr, err := json.Marshal(jsonMap)
	if err != nil {
		return nil, err
	}

	return jsonStr, nil
}

func switchContext(context, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
			&clientcmd.ConfigOverrides{
					CurrentContext: context,
			}).ClientConfig()
}
