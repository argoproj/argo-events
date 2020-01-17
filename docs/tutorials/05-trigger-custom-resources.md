# Trigger Custom Resources
Argo Events supports Argo workflows, standard K8s objects, Sensor and Gateway as
K8s triggers. In order to add your custom resource as trigger to Argo Events, you will
need to register the scheme of resource in Argo Events.

If you feel Argo Events should support your custom resource out of box, create an issue
on GitHub and provide the details.

## Steps to build sensor image with your CR

1. Fork the Argo Events project.
2. Go to store.go in store package.
3. Import your custom resource api package.
4. In init method, add the scheme to your custom resource api.
5. Make sure there are no errors.
6. Rebuild the sensor binary using make sensor
7. To build the image, first change IMAGE_NAMESPACE in Makefile to your docker registry and then run make sensor-image.
