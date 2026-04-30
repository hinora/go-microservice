# Core Concepts

## Broker

The broker is the local runtime for a node. It owns the service registry, routes calls, starts discovery and transport, records tracing and metrics, and applies middleware and resilience policies.

## Node

A node is one running broker instance identified by `BrokerConfig.NodeId`. In multi-node deployments, each node announces itself through discovery and routes remote traffic through the transporter.

## Service

A service groups actions and events under one name. A service can also define lifecycle hooks, dependencies, mixins, versioning, and service-wide action hooks.

## Action

An action is a request/response handler addressed as `service.action`. Versioned services are addressed as `v{version}.service.action` and the unversioned address resolves to the latest registered version.

## Event

An event is a fire-and-forget handler addressed by event name. Calling an event through `ctx.Call` uses group-based balancing, while `ctx.Broadcast` or `broker.Broadcast` sends to every matching subscriber.

## Context

Every action and event receives a `*goservice.Context` containing params, metadata, caller information, tracing IDs, nested call helpers, broadcast helpers, logging helpers, and the owning service.

## Registry and discovery

The registry tracks nodes, services, actions, events, REST metadata, and action timeouts. Redis discovery exchanges this information and heartbeats between nodes.

## Transporter

The transporter moves requests, responses, and events between nodes. Redis is the currently supported transporter.
