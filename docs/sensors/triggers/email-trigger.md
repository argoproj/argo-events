# Email Trigger

The Email trigger is used to send a custom email to a desired set of email addresses using an SMTP server. The intended use is for notifications for a build pipeline, but can be used for any notification scenario.

## Prerequisite

1.  Deploy the eventbus in the namespace.

2.  Have an SMTP server setup.

3.  Create a kubernetes secret with the SMTP password in your cluster.

        kubectl create secret generic smtp-secret --from-literal=password=$SMTP_PASSWORD

    **Note**: If your SMTP server doesnot require authentication this step can be skipped.

4.  Create a webhook event-source.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml

5.  Set up port-forwarding to expose the http server. We will
    use port-forwarding here.

         kubectl port-forward -n argo-events <event-source-pod-name> 12000:12000

## Email Trigger

Lets say we want to send an email to a dynamic recipient using a custom email body template.

The custom email body template we are going to use is the following:

```
Hi <name>,
  Hello There

Thanks,
Obi
```

where the name has to be substituted with the receiver name from the event.

1.  Create a sensor with Email trigger.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/email-trigger.yaml

    **Note**: Please update `email.port`, `email.host` and `email.username` to that of your SMTP server.
    If your SMTP server doesnot require authentication, the `email.username` and `email.smtpPassword` should be omitted.

2.  Send a http request to the event-source-pod to fire the Email trigger.

        curl -d '{"name":"Luke", "to":"your@email.com"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

    **Note**: You can modify the value for key `"to"` to send the email to your address.

3.  Alternatively you can skip providing the `"to"` in the payload to send an email to static email address provided in the trigger.

        curl -d '{"name":"Luke"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

    **Note**: You have to remove the parameterization for `email.to.0` and add `email.to` like so:
    `yaml
    email:
      ...
      to:
        - target1@email.com
        - target2@email.com
      ...
    `

## Parameterization

We can parameterize the to, from, subject and body of the email trigger for dynamic capabilities.

The email trigger parameters have the following structure,

    - parameters:
        - src:
            dependencyName: test-dep
            dataKey: body.to
          dest: email.to.0
        - src:
            dependencyName: test-dep
            dataKey: body.to
          dest: email.to.-1
        - src:
            dependencyName: test-dep
            dataKey: body.from
          dest: email.from
        - src:
            dependencyName: test-dep
            dataKey: body.subject
          dest: email.subject
        - src:
            dependencyName: test-dep
            dataKey: body.emailBody
          dest: email.body

- `email.to.index` can be used to overwrite an email address already specified in the trigger at the provided index. (where index is an integer)
- `email.to.-1` can be used to append a new email address to the addresses to which an email will be sent.
- `email.from` can be used to specify the from address of the email sent.
- `email.body` can be used to specify the body of the email which will be sent.
- `email.subject` can be used to specify the subject of the email which will be sent.

To understand more on parameterization, take a look at [this tutorial](https://argoproj.github.io/argo-events/tutorials/02-parameterization/).

The complete specification of Email trigger is available [here](../../APIs.md#argoproj.io/v1alpha1.EmailTrigger).
