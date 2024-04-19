#!/bin/bash
set -xeo pipefail

ns="openshift-ovn-kubernetes"
mlabel="app=ovnkube-master"
nlabel="app=ovnkube-node"

for pod_name in $(oc get pods -n $ns -l $mlabel  --output jsonpath="{range .items[*].metadata}{.name}{'\n'}{end}"); do
  mkdir -p "/tmp/histodata/master/$pod_name/"
  oc -n $ns cp -c ovnkube-master $pod_name:/tmp /tmp/histodata/master/$pod_name
done

for pod_name in $(oc get pods -n $ns -l $nlabel  --output jsonpath="{range .items[*].metadata}{.name}{'\n'}{end}"); do
  mkdir -p "/tmp/histodata/master/$pod_name/"
  oc -n $ns cp -c ovnkube-node $pod_name:/tmp /tmp/histodata/node/$pod_name
done

tar -czvf histo-data.tar.gz /tmp/histodata
rm -rf /tmp/histodata

echo "done - tar file called histo-data.tar.gz generated "
