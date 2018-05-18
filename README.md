# k8s-nodelabels-fswriter

This small tool permits to get node labels near the kubrenetes apiserver, it acts as a watcher.
It can write obtained labels to a file in a json format.

Initial need was to retrieve information about a node like a TOR ip and an ASN, to be able to peer with TOR from a pod running a gobgp container with its own network namespace.
