#!/usr/bin/env sh

pandoc --from markdown --to gfm event-source.html > event-source.md
pandoc --from markdown --to gfm gateway.html > gateway.md
pandoc --from markdown --to gfm sensor.html > sensor.md
