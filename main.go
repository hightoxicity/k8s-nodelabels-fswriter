package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	api "k8s.io/kubernetes/pkg/apis/core"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig     = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	nodename       = flag.String("nodename", "", "node name")
	destfile       = flag.String("o", "", "file output")
	fOutputSetLock = sync.Mutex{}
)

func NodeLabelsToJson(nd *v1.Node) (string, error) {
	jsonBuf, err := json.Marshal(nd.ObjectMeta.Labels)
	if err != nil {
		return "", err
	}

	return string(jsonBuf), nil
}

func WriteToFile(destFile string, content string) error {
	fp, ferr := os.Create(destFile)

	if ferr != nil {
		return ferr
	}

	defer fp.Close()
	fp.WriteString(content)

	return nil
}

func GetMyFqdn() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}

	addrs, err := net.LookupIP(hostname)
	if err != nil {
		return hostname
	}

	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			ip, err := ipv4.MarshalText()
			if err != nil {
				return hostname
			}
			hosts, err := net.LookupAddr(string(ip))
			if err != nil || len(hosts) == 0 {
				return hostname
			}
			fqdn := hosts[0]
			return strings.TrimSuffix(fqdn, ".") // return fqdn without trailing dot
		}
	}
	return hostname
}

func ProcessNodeLabels(destFile string, nd *v1.Node) {
	json, err := NodeLabelsToJson(nd)
	if err == nil {

		if destFile == "" {
			fmt.Println(json)
		} else {
			doWrite := true

			fOutputSetLock.Lock()
			defer fOutputSetLock.Unlock()

			file, err := os.Open(destFile)
			if err == nil {
				defer file.Close()
				tablePolynomial := crc32.MakeTable(0xedb88320)
				hash := crc32.New(tablePolynomial)

				if _, err := io.Copy(hash, file); err == nil {
					oldHashInBytes := hash.Sum(nil)[:]

					tablePolynomial := crc32.MakeTable(0xedb88320)
					hash := crc32.New(tablePolynomial)
					if _, err := io.Copy(hash, strings.NewReader(json)); err == nil {
						file.Close()
						newHashInBytes := hash.Sum(nil)[:]

						if bytes.Compare(oldHashInBytes, newHashInBytes) == 0 {
							log.Println("Labels did not changed")
							doWrite = false
						}
					}
				}
			}

			if doWrite {
				log.Println("Labels need to be written")
				ferr := WriteToFile(destFile, json)

				if ferr != nil {
					log.Fatalf("Cannot write to file %s: %s", destFile, ferr)
				}
			}
		}
	}
}

func main() {
	flag.Parse()

	var err error
	var config *rest.Config

	if *kubeconfig == "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Fatalf("Error with config file %s: %s", *kubeconfig, err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Bad config file: %s", err)
	}

	ndNameToUse := string(*nodename)

	if ndNameToUse == "" {
		ndNameToUse = os.Getenv("HOSTNAME")
		if ndNameToUse == "" {
			ndNameToUse = GetMyFqdn()
		}
	}

	log.Printf("We will watch labels of following node: %s", ndNameToUse)

	fieldSelector := fields.Set{api.ObjectNameField: ndNameToUse}.AsSelector()

	watchlist := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"nodes",
		v1.NamespaceAll,
		fieldSelector,
	)
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Node{},
		0, //Duration is int64
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ProcessNodeLabels(*destfile, obj.(*v1.Node))
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				ProcessNodeLabels(*destfile, newObj.(*v1.Node))
			},
		},
	)
	stop := make(chan struct{})
	defer close(stop)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go controller.Run(stop)
	<-sigs
}
