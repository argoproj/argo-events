# Trigger Conditions

> v1.0 and after

Triggers can be executed based on different dependency `conditions`.

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

## Conditions Reset

When multiple dependencies are defined for a trigger, the trigger won't be executed until the condition expression is resolved to `true`. Sometimes you might want to reset all the stakeholders of the conditions, `conditions reset` is the way to do it.

For example, your trigger has a condition as `A && B`, both `A` and `B` are expected to have an event everyday. One day for some reason, `A` gets an event but `B` doesn't, then it ends up with today's `A` and tomorrow's `B` triggering an action, which might not be something you want. To avoid that, you can reset the conditions as following:

```yaml
spec:
  triggers:
    - template:
        conditions: "dep01 && dep02"
        conditionsReset:
          - byTime:
              # Reset conditions at 23:59
              cron: "59 23 * * *"
              # Optional, defaults to UTC
              # More info for timezone: https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
              timezone: America/Los_Angeles
        name: trigger01
```
