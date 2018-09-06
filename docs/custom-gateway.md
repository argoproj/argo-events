# Custom Gateway

You can write a gateway in 3 different ways,

1. Implement in core gateway style.
2. Implement a gRPC server in any language of your choice.
3. Write your own implementation completely independent of framework.

Difference between first two options and third is that in both option 1 and 2, framework provides a mechanism
to watch configuration updates, start/stop a configuration dynamically. In option 3, its up to
user to watch configuration updates and take actions.

Let's start with first option.

## Core Gateway Style
It is the most straightforward option. User needs to  