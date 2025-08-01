# Contributing

## How To Provide Feedback

Please [raise an issue in Github](https://github.com/argoproj/argo-events/issues).

## Code of Conduct

See [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

## Contributor Meetings

A weekly opportunity for committers and maintainers of Workflows, Events, and Dataflow to discuss their current work and
talk about what’s next. Feel free to join us! For Contributor Meeting information, minutes and recordings
please [see here](https://bit.ly/argo-data-weekly).

## How To Contribute

We're always looking for contributors.

- Documentation - something missing or unclear? Please submit a pull request!
- Code contribution - investigate
  a [good first issue](https://github.com/argoproj/argo-events/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22)
  , or anything not assigned.
- Join the `#argo-contributors` channel on [our Slack](https://argoproj.github.io/community/join-slack).

### Running Locally

To run Argo Events locally for development: [developer guide](developer_guide.md).

### Dependencies

Dependencies increase the risk of security issues and have on-going maintenance costs.

The dependency must pass these test:

- A strong use case.
- It has an acceptable license (e.g. MIT).
- It is actively maintained.
- It has no security issues.

Example, should we add `fasttemplate`
, [view the Snyk report](https://snyk.io/advisor/golang/github.com/valyala/fasttemplate):

| Test                                    | Outcome                              |
| --------------------------------------- | ------------------------------------ |
| A strong use case.                      | ❌ Fail. We can use `text/template`. |
| It has an acceptable license (e.g. MIT) | ✅ Pass. MIT license.                |
| It is actively maintained.              | ❌ Fail. Project is inactive.        |
| It has no security issues.              | ✅ Pass. No known security issues.   |

No, we should not add that dependency.

### Contributor Workshop

We have a [90m video on YouTube](https://youtu.be/zZv0lNCDG9w) to show you how to get some hands-on experience contributing (e.g., running locally, testing, etc...).
