# Shared configuration

This directory holds non-secret contracts shared by the API and the scanner.

Secrets stay in the environment. Copy `.env.example` to `.env` and replace the passwords there. Do not commit `.env`.

`queues.yaml` names the Redis lists the API and the scanner will use once job execution exists. Both services default to these names. Neither service consumes those lists yet.
