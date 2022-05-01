# Sensor High Availability

Sensor controller creates a k8s deployment (replica number defaults to 1) for
each Sensor object. HA with `Active-Passive` strategy can be achieved by setting
`spec.replicas` to an odd number greater than 2, which means only one Pod serves
traffic and the rest stand by. One of standby Pods will be automatically
elected to be active if the old one is gone. Note that an odd number is recommended
since leader election uses the RAFT algorithm, which recommends that for consensus.

**Please DO NOT manually scale up the replicas, that might cause unexpected
behaviors!**

Click [here](../dr_ha_recommendations.md) to learn more information about Argo
Events DR/HA recommendations.
