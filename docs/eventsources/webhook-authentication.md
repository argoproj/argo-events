# Webhook Authentication

![GA](../assets/ga.svg)

> v1.0 and after

For `webhook` event source, if you want to get your endpoint protected from
unauthorized accessing, you can specify `authSecret` to the spec, which is a K8s
secret key selector.

This simple authentication approach also works for `webhook` extended event
sources, if that event source does not have a built in authenticator.

Firstly, create a k8s secret containing your token.

```sh
echo -n 'af3qqs321f2ddwf1e2e67dfda3fs' > ./token.txt

kubectl create secret generic my-webhook-token --from-file=my-token=./token.txt
```

Then add `authSecret` to your `webhook` EventSource.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: webhook
spec:
  webhook:
    example:
      port: "12000"
      endpoint: /example
      method: POST
      authSecret:
        name: my-webhook-token
        key: my-token
```

Now you can authenticate your webhook endpoint with the configured token.

```sh
TOKEN="Bearer af3qqs321f2ddwf1e2e67dfda3fs"

curl -X POST -H "Authorization: $TOKEN" -d "{your data}" http://xxxxx:12000/example
```
