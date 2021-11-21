package utils

import (
	"fmt"

	"github.com/lithammer/dedent"
	"k8s.io/klog/v2"
)

/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

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

const (
	kArmadaAgent = `---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: karmada-agent
rules:
  - apiGroups: ['*']
    resources: ['*']
    verbs: ["get", "watch", "list", "create", "update", "delete"]
  - nonResourceURLs: ['*']
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: karmada-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: karmada-agent
subjects:
  - kind: ServiceAccount
    name: karmada-agent-sa
    namespace: karmada-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: karmada-agent-sa
  namespace: karmada-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: karmada-agent
  namespace: karmada-system
  labels:
    app: karmada-agent
spec:
  replicas: 1
  selector:
    matchLabels:
      app: karmada-agent
  template:
    metadata:
      labels:
        app: karmada-agent
    spec:
      serviceAccountName: karmada-agent-sa
      tolerations:
        - key: node-role.kubernetes.io/master
          operator: Exists
      containers:
        - name: karmada-agent
          image: swr.ap-southeast-1.myhuaweicloud.com/karmada/karmada-agent:latest
          command:
            - /bin/karmada-agent
            - --karmada-kubeconfig=/etc/kubeconfig/karmada-kubeconfig
            - --karmada-context=karmada
            - --cluster-name={member_cluster_name}
            - --cluster-status-update-frequency=10s
            - --v=4
          volumeMounts:
            - name: kubeconfig
              mountPath: /etc/kubeconfig
      volumes:
        - name: kubeconfig
          secret:
            secretName: karmada-kubeconfig`

	estimator = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: karmada-scheduler-estimator-{{member_cluster_name}}
  namespace: karmada-system
  labels:
    cluster: {{member_cluster_name}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: karmada-scheduler-estimator-{{member_cluster_name}}
  template:
    metadata:
      labels:
        app: karmada-scheduler-estimator-{{member_cluster_name}}
    spec:
      tolerations:
        - key: node-role.kubernetes.io/master
          operator: Exists
      containers:
        - name: karmada-scheduler-estimator
          image: swr.ap-southeast-1.myhuaweicloud.com/karmada/karmada-scheduler-estimator:latest
          imagePullPolicy: IfNotPresent
          command:
            - /bin/karmada-scheduler-estimator
            - --kubeconfig=/etc/{{member_cluster_name}}-kubeconfig
            - --cluster-name={{member_cluster_name}}
          volumeMounts:
            - name: member-kubeconfig
              subPath: {{member_cluster_name}}-kubeconfig
              mountPath: /etc/{{member_cluster_name}}-kubeconfig
      volumes:
        - name: member-kubeconfig
          secret:
            secretName: {{member_cluster_name}}-kubeconfig
---
apiVersion: v1
kind: Service
metadata:
  name: karmada-scheduler-estimator-{{member_cluster_name}}
  namespace: karmada-system
  labels:
    cluster: {{member_cluster_name}}
spec:
  selector:
    app: karmada-scheduler-estimator-{{member_cluster_name}}
  ports:
    - protocol: TCP
      port: 10352
      targetPort: 10352`
)

func GenExamples(path string) {

	//
	if err := BytesToFile(path, "karmada-agent.yaml", []byte(kArmadaAgent)); err != nil {
		klog.Warning(err)
	}

	// https://github.com/karmada-io/karmada/blob/master/artifacts/deploy/karmada-scheduler-estimator.yaml
	if err := BytesToFile(path, "karmada-scheduler-estimator.yaml", []byte(estimator)); err != nil {
		klog.Warning(err)
	}

	fmt.Println(dedent.Dedent(`┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
| Push mode                                                                                                                                                           |
|                                                                                                                                                                     |
| Step 1: Member kubernetes join karmada control plane                                                                                                                |
|                                                                                                                                                                     |
| (In karmada)~#  cat ~/.kube/config  | grep current-context | sed 's/: /\n/g'| sed '1d' #MEMBER_CLUSTER_NAME                                                         |
| (In karmada)~# kubectl-karmada  --kubeconfig /var/lib/karmada/karmada-apiserver.config  join ${MEMBER_CLUSTER_NAME} --cluster-kubeconfig=$HOME/.kube/config         |
|                                                                                                                                                                     |
| Step 2: Create member kubernetes kubeconfig secret                                                                                                                  |
|                                                                                                                                                                     |
| (In member kubernetes)~# kubectl create ns karmada-system                                                                                                           |
| (In member kubernetes)~# kubectl create secret generic ${MEMBER_CLUSTER_NAME}-kubeconfig --from-file=${MEMBER_CLUSTER_NAME}-kubeconfig=$HOME/.kube/config  -n karmada-system              |
|                                                                                                                                                                     |
| Step 3: Create karmada scheduler estimator                                                                                                                          |
|                                                                                                                                                                     |
| (In member kubernetes)~# sed -i "s/{{member_cluster_name}}/${MEMBER_CLUSTER_NAME}/g" /var/lib/karmada/karmada-scheduler-estimator.yaml                              |
| (In member kubernetes)~# kubectl create -f  /var/lib/karmada/karmada-scheduler-estimator.yaml                                                                       |
|                                                                                                                                                                     |
| Step 4: Show members of karmada                                                                                                                                     |
|                                                                                                                                                                     |
| (In karmada)~# kubectl  --kubeconfig /var/lib/karmada/karmada-apiserver.config get clusters                                                                         |
|                                                                                                                                                                     |
├── —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —— —──┤
| Pull mode                                                                                                                                                           |
|                                                                                                                                                                     |
| Step 1:  Send karmada kubeconfig and karmada-agent.yaml to member kubernetes                                                                                        |
|                                                                                                                                                                     |
| (In karmada)~# scp /var/lib/karmada/karmada-apiserver.config /var/lib/karmada/karmada-agent.yaml {member kubernetes}:~                                              |
|                                                                                                                                                                     |
| Step 2:  Create karmada kubeconfig secret                                                                                                                           |
|  Notice:                                                                                                                                                            |
|    Cross-network, need to change the config server address.                                                                                                         |
|                                                                                                                                                                     |
| (In member kubernetes)~#  kubectl create ns karmada-system                                                                                                          |
| (In member kubernetes)~#  kubectl create secret generic karmada-kubeconfig --from-file=karmada-kubeconfig=/root/karmada-apiserver.config  -n karmada-system         |
|                                                                                                                                                                     |
| Step 3: Create karmada agent                                                                                                                                        |
|                                                                                                                                                                     |
| (In member kubernetes)~#  MEMBER_CLUSTER_NAME="demo"                                                                                                                |
| (In member kubernetes)~#  sed -i "s/{member_cluster_name}/${MEMBER_CLUSTER_NAME}/g" karmada-agent.yaml                                                              |
| (In member kubernetes)~#  kubectl create -f karmada-agent.yaml                                                                                                      |
|                                                                                                                                                                     |
| Step 4: Show members of karmada                                                                                                                                     |
|                                                                                                                                                                     |
| (In karmada)~# kubectl  --kubeconfig /var/lib/karmada/karmada-apiserver.config get clusters                                                                         |
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
`))
}
