local env = std.extVar("__ksonnet/environments");
local params = std.extVar("__ksonnet/params").components["argo-events"];
local k = import "k.libsonnet";
local deployment = k.apps.v1beta1.deployment;
local container = k.apps.v1beta1.deployment.mixin.spec.template.spec.containersType;
local configmap = k.core.v1.configMap;
local crd = k.apiextensions.v1beta1.customResourceDefinition;

local labels = {app: params.controllerName};

local sensor = crd.new();

local configString = 
"namespace: %s
serviceAccount: %s
executorImage: %s
executorResources:
    limits:
        cpu: %s
        mem: %s
    requests:
        cpu: %s
        mem: %s
";
local configData = {
    "namespace": params.namespace,
    "serviceAccount": params.serviceAccount,
    "executorImage": params.executorImage,
    "executorResources": params.executorResources,
};

local configMapName = params.controllerName + "-configmap";

local configMap = configmap
    .new(
        configMapName,
        std.manifestYamlDoc(configData));

local controllerDeployment = deployment
    .new(
        params.controllerName,
        params.replicas,
        container
            .new(params.controllerName, params.controllerImage)
            .withEnv([container.envType.new("SENSOR_NAMESPACE", params.namespace),
            container.envType.new("SENSOR_CONFIG_MAP", configMapName)]))
    .withNamespace(params.namespace)
    .withServiceAccount(params.serviceAccount);

k.core.v1.list.new([configMap, controllerDeployment])