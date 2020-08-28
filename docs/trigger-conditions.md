# Trigger Conditions

![alpha](assets/alpha.svg)

> v1.0 and after

`Conditions` is a new feature to replace `Circuit` and `Switch`. With
`conditions`, triggers can be executed based on different dependency conditions.

An example with `conditions`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: example
spec:
  dependencies:
    - name: dep01
      eventSourceName: webhook-a
      eventName: example01
    - name: dep02
      eventSourceName: webhook-a
      eventName: example02
   - name: dep03
      eventSourceName: webhook-b
      eventName: example03
  triggers:
    - template:
        conditions: "dep02"
        name: trigger01
        http:
          url: http://abc.com/hello1
          method: GET
    - template:
        conditions: "dep02 && dep03"
        name: trigger02
        http:
          url: http://abc.com/hello2
          method: GET
    - template:
        conditions: "(dep01 || dep02) && dep03"
        name: trigger03
        http:
          url: http://abc.com/hello3
          method: GET
```

`Conditions` is a boolean expression contains dependency names, the trigger
won't be executed until the expression resolves to true. The operators in
`conditions` include:

- `&&`
- `||`

## Triggers Without Conditions

If `conditions` is missing, the default conditions to execute the trigger is
`&&` logic of all the defined dependencies.
