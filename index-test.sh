#!/bin/bash

go run cmd/patman/main.go \
  -file logs.txt \
  -index trace_id \
  -format json \
  'r:"/ |> m:trace_id:(\s+)?\d+ |> m:\d+ |> name:trace_id' \
  'r:"/ |> m:amount:(\s+)\d+ |> m:\d+ |> name:amount' \
  'm:"user":(\s+)".*" |> r:"/ |> r:user:/ |> name:user'