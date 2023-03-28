FROM registry.com/builder:latest@sha256:d23df29669d05462cf55ce2274a3a897aa2e2655d0fad104375f8ef06164b575 as builder

FROM registry.com/example
LABEL name="example"
