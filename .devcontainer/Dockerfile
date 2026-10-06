FROM mcr.microsoft.com/devcontainers/python:1-3.12-bookworm

USER root

# Remove the unused Yarn repository with the broken signing key.
RUN rm -f /etc/apt/sources.list.d/yarn.list