
# Time Filter

You can use time filter, which is applied on event time.
It filters out events that occur outside the specified time range, so it is specially helpful when
you need to make sure an event occurs between a certain time-frame.

Time filter takes a `start` and `stop` time in `HH:MM:SS` format in UTC. If `stop` is smaller than `start`,
the stop time is treated as next day of `start`. Note that `start` is inclusive while `stop` is exclusive.
The diagrams below illustlate these behavior.

An example of time filter is available under `examples/sensors`.

1. if `start` < `stop`: event time must be in `[start, stop)`.

         00:00:00                            00:00:00                            00:00:00
         ┃     start                   stop  ┃     start                   stop  ┃
        ─┸─────●───────────────────────○─────┸─────●───────────────────────○─────┸─
               ╰───────── OK ──────────╯           ╰───────── OK ──────────╯

1. if `stop` < `start`: event time must be in `[start, stop@Next day)`  
   (this is equivalent to: event time must be in `[00:00:00, stop) || [start, 00:00:00@Next day)`).

         00:00:00                            00:00:00                            00:00:00
         ┃           stop        start       ┃       stop            start       ┃
        ─┸───────────○───────────●───────────┸───────────○───────────●───────────┸─
        ─── OK ──────╯           ╰───────── OK ──────────╯           ╰────── OK ───

## Further examples

You can find some examples [here](https://github.com/argoproj/argo-events/tree/master/examples/sensors).
