#!/bin/bash

go run cmd/patman/main.go \
  -file ./logs.txt \
  -format csv \
  'r:"|\s/ |> m:trace_id:\d+ |> r:trace_id:/ |> name:trace_id' \
  'ml:amount |> r:"|:|\s+/ |> m:amount\d+ |> r:amount/ |> name:amount' \
  'r:"/ |> m:user:\s+\w+ |> r:user:\s/ |> name:user'
