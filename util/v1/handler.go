package v1

import (
	"encoding/json"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	extensionsbeta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	. "kappagent/global"
	"kappagent/util"
	"kappagent/util/common"
	"time"
)

func GetResourceWithNamespace(clientSet *kubernetes.Clientset) []NamespaceOld {
	util.Log.Info("正在获取项目数据...")
	var ns []NamespaceOld

	namespaceItems, _ := clientSet.CoreV1().Namespaces().List(metav1.ListOptions{})
	nitems := namespaceItems.Items

	for i := range nitems {
		// 收集deployment
		nname := nitems[i].Name
		if nname == "default" || nname == "kube-system" || nname == "kube-public" ||
			nname == "local" || nname == "tools" || RegExp.MatchString(nname) {
			continue
		}
		var ss []StatefulSetOld
		var ds []DeploymentOld

		deploymentsClient, _ := clientSet.ExtensionsV1beta1().Deployments(nname).List(metav1.ListOptions{})
		ditems := deploymentsClient.Items

		if len(ditems) == 0 {
			util.Log.Infof("namespace: %s has no deployment", nname)
		} else {
			for q := range ditems {
				o := ditems[q]

				ps := common.GetPod(clientSet, nname, o.Spec.Selector.MatchLabels)
				ds = append(ds, DeploymentOld{Data: o, Pods: ps})
			}
		}

		// 收集statefulset
		statefulsetsClient, _ := clientSet.AppsV1beta1().StatefulSets(nname).List(metav1.ListOptions{})
		sitems := statefulsetsClient.Items
		if len(sitems) == 0 {
			util.Log.Infof("namespace: %s has no statefulsets", nname)
		} else {
			for q := range sitems {
				o := sitems[q]

				ps := common.GetPod(clientSet, nname, o.Spec.Selector.MatchLabels)
				ss = append(ss, StatefulSetOld{Data: o, Pods: ps})
			}
		}

		ns = append(ns, NamespaceOld{Name: nname, Deployments: ds, StatefulSets: ss})
	}
	util.Log.Info("获取项目数据完成...")
	return ns
}

func WatchDepHandler(clientSet *kubernetes.Clientset, watchDeploymentChannel chan WatchOldDepData) error {
	util.Log.Info("正在监听deployment...")
	deploymentsClient := clientSet.ExtensionsV1beta1().Deployments(metav1.NamespaceAll)

	list, _ := deploymentsClient.List(metav1.ListOptions{})
	items := list.Items

	timeoutSeconds := int64((15 * time.Minute).Seconds())
	options := metav1.ListOptions{
		TimeoutSeconds: &timeoutSeconds,
	}
	w, _ := deploymentsClient.Watch(options)
	defer w.Stop()

	// 为了第一次不发送数据，启动watch第一次会输出所有的数据
	count := 0
	// watch有超时时间，如果不在listoption里面设置TimeoutSeconds，默认30到60分钟会断开链接，
	// 所以用ok来监视是否断开链接
loop:
	for {
		select {
		case e, ok := <-w.ResultChan():
			if !ok {
				break loop
			} else if e.Type == watch.Added || e.Type == watch.Deleted || e.Type == watch.Modified {
				if count != len(items) {
					count += 1
				} else {
					// go的断言获取运行时的struct
					nname := e.Object.(*extensionsbeta1.Deployment).Namespace
					if nname != "default" && nname != "kube-system" &&
						nname != "kube-public" && nname != "local" && nname != "tools" &&
						!RegExp.MatchString(nname) {
						data := WatchOldDepData{
							Deployment: e.Object.(*extensionsbeta1.Deployment),
							Namespace:  e.Object.(*extensionsbeta1.Deployment).Namespace,
							Type:       e.Type,
						}
						watchDeploymentChannel <- data
					}
				}
			}
		}
	}
	return nil
}

func WatchStatefulHandler(clientSet *kubernetes.Clientset, watchStatefulSetChannel chan WatchOldStatefulData) error {
	util.Log.Info("正在监听statefulset...")
	statefulSetClient := clientSet.AppsV1beta1().StatefulSets(metav1.NamespaceAll)

	list, _ := statefulSetClient.List(metav1.ListOptions{})
	items := list.Items

	timeoutSeconds := int64((15 * time.Minute).Seconds())
	options := metav1.ListOptions{
		TimeoutSeconds: &timeoutSeconds,
	}
	w, _ := statefulSetClient.Watch(options)
	defer w.Stop()

	// 为了第一次不发送数据，启动watch第一次会输出所有的数据
	count := 0
	// watch有超时时间，如果不在listoption里面设置TimeoutSeconds，默认30到60分钟会断开链接，
	// 所以用ok来监视是否断开链接
loop:
	for {
		select {
		case e, ok := <-w.ResultChan():
			if !ok {
				break loop
			} else if e.Type == watch.Added || e.Type == watch.Deleted || e.Type == watch.Modified {
				if count != len(items) {
					count += 1
				} else {
					// go的断言获取运行时的struct
					nname := e.Object.(*v1beta1.StatefulSet).Namespace
					if nname != "default" && nname != "kube-system" &&
						nname != "kube-public" && nname != "local" && nname != "tools" &&
						!RegExp.MatchString(nname) {
						data := WatchOldStatefulData{
							StatefulSet: e.Object.(*v1beta1.StatefulSet),
							Namespace:   e.Object.(*v1beta1.StatefulSet).Namespace,
							Type:        e.Type,
						}
						watchStatefulSetChannel <- data
					}
				}
			}
		}
	}
	return nil
}
func WatchNodeHandler(clientSet *kubernetes.Clientset, watchNodeChannel chan WatchNodeData) error {
	util.Log.Info("正在监听node...")
	nodesClient := clientSet.CoreV1().Nodes()

	list, _ := nodesClient.List(metav1.ListOptions{})
	items := list.Items

	timeoutSeconds := int64((15 * time.Minute).Seconds())
	options := metav1.ListOptions{
		TimeoutSeconds: &timeoutSeconds,
	}
	w, _ := nodesClient.Watch(options)
	defer w.Stop()

	// 为了第一次不发送数据，启动watch第一次会输出所有的数据
	count := 0
	// watch有超时时间，如果不在listoption里面设置TimeoutSeconds，默认30到60分钟会断开链接，
	// 所以用ok来监视是否断开链接
loop:
	for {
		select {
		case e, ok := <-w.ResultChan():
			if !ok {
				break loop
			} else if e.Type == watch.Added || e.Type == watch.Deleted{
				if count != len(items) {
					count += 1
				} else {
					data := WatchNodeData{
						Node: e.Object.(*v1.Node),
						Type: e.Type,
					}
					watchNodeChannel <- data
				}
			}
		}
	}
	return nil
}

// 接收channel发送数据
func GetChannel(clientSet *kubernetes.Clientset) {
	for {
		select {
		case e := <-WatchDeploymentChannel:
			util.Log.Infof("%s Deployment,Name: %s,NameSpace: %s", e.Type, e.Deployment.Name, e.Namespace)
			watchProject := &WatchProjectOld{
				ClusterName:  ClusterName,
				Type:         e.Type,
				Timestamp:    time.Now().Unix(),
				ResourceType: "Deployment",
				Namespaces: []NamespaceOld{
					{
						Name: e.Namespace,
						Deployments: []DeploymentOld{
							{
								Data: *e.Deployment,
								Pods: common.GetPod(clientSet, e.Namespace, e.Deployment.Spec.Selector.MatchLabels),
							},
						},
					},
				},
			}

			jsonBytes, err := json.Marshal(watchProject)
			if err != nil {
				util.Log.Error(err)
			}

			util.HttpPostForm(string(jsonBytes), SiteUrl)
		case e := <-WatchStatefulSetChannel:
			util.Log.Infof("%s StatefulSet,Name: %s,NameSpace: %s", e.Type, e.StatefulSet.Name, e.Namespace)
			watchProject := &WatchProjectOld{
				ClusterName:  ClusterName,
				Type:         e.Type,
				Timestamp:    time.Now().Unix(),
				ResourceType: "StatefulSet",
				Namespaces: []NamespaceOld{
					{
						Name: e.Namespace,
						StatefulSets: []StatefulSetOld{
							{
								Data: *e.StatefulSet,
								Pods: common.GetPod(clientSet, e.Namespace, e.StatefulSet.Spec.Selector.MatchLabels),
							},
						},
					},
				},
			}

			jsonBytes, err := json.Marshal(watchProject)
			if err != nil {
				util.Log.Error(err)
			}

			util.HttpPostForm(string(jsonBytes), SiteUrl)
		case e := <-WatchNodeChannel:
			util.Log.Infof("%s Node,Addresses: %s", e.Type, e.Node.Status.Addresses)
			watchNode := &WatchNode{
				ClusterName:  ClusterName,
				Type:         e.Type,
				Timestamp:    time.Now().Unix(),
				ResourceType: "Node",
				Node:         *e.Node,
			}

			jsonBytes, err := json.Marshal(watchNode)
			if err != nil {
				util.Log.Error(err)
			}

			util.HttpPostForm(string(jsonBytes), SiteUrl)
		}
	}
}