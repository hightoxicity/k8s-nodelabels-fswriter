#!/bin/bash

kubectl apply -f ./k8s-defs/cluster-role/nodelabels-reader.yaml 
kubectl apply -f ./k8s-defs/cluster-role-binding/kube-system-fswriter-nodelabels-reader.yaml
kubectl apply -f ./k8s-defs/daemon-set/k8s-nodelabels-fswriter.yaml
kubectl apply -f ./k8s-defs/service-account/nodelabels-fswriter.yaml
