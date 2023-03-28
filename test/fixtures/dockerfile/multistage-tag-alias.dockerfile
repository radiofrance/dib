FROM registry.com/builder:latest as builder

FROM registry.com/example
LABEL name="example"
