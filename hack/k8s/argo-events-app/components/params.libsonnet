{
  global: {
    // User-defined global parameters; accessible to all component and environments, Ex:
    // replicas: 4,
  },
  components: {
    // Component-level parameters, defined initially from 'ks prototype use ...'
    // Each object below should correspond to a component in the components/ directory
    "argo-events": {
      serviceAccount: "argo-events",
      namespace: "default",
      replicas: 1,
      controllerName: "gateway-controller",
      controllerImage: "argoproj/gateway-controller:latest",
      executorName: "gateway-executor",
      executorImage: "argoproj/executor-job:latest",
      executorResources: {
        limits: {
          cpu: "150m",
          mem: "100Mi",
        },
        requests: {
          cpu: "50m",
          mem: "100Mi",
        },
      },
    },
  },
}
