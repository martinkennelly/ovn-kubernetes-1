#!/bin/bash
set -eo pipefail
set -x

OVNK="/home/mkennell/repos/ovn-kubernetes/dist/images"

[ -z "$1" ] && echo "need tag" && exit 1

tag=$1
echo "tag used: $tag"
cd "$OVNK"
make fedora
docker tag ovn-kube-f:latest quay.io/mkennell/ovn:$tag
docker push quay.io/mkennell/ovn:$tag


# sdn
oc patch clusterversion version --type json -p '[{"op":"add","path":"/spec/overrides","value":[{"kind":"Deployment","group":"apps","name":"network-operator","namespace":"openshift-network-operator","unmanaged":true}]}]'
oc -n openshift-network-operator delete deployment network-operator || true
oc -n openshift-ovn-kubernetes set image ds/ovnkube-node ovnkube-controller=quay.io/mkennell/ovn:$tag
oc -n openshift-ovn-kubernetes set image deployment/ovnkube-control-plane ovnkube-cluster-manager=quay.io/mkennell/ovn:$tag
oc get pods -n openshift-ovn-kubernetes
