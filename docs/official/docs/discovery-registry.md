# Discovery Registry

Discovery lets nodes announce themselves and share their service registry over Redis.

```go
DiscoveryConfig: goservice.DiscoveryConfig{
    Enable:        true,
    DiscoveryType: goservice.DiscoveryTypeRedis,
    Config: goservice.DiscoveryRedisConfig{
        Host:     "127.0.0.1",
        Port:     6379,
        Password: "",
        Db:       0,
    },
    HeartbeatInterval:        3000,
    HeartbeatTimeout:         7000,
    CleanOfflineNodesTimeout: 9000,
},
```

## Registry contents

Each node keeps registry entries for:

- node ID and IP addresses
- service names and optional versions
- actions, REST metadata, params hints, and action timeouts
- events and params hints

## Lifecycle

1. A node starts its broker and local registry.
2. The node broadcasts discovery information over Redis.
3. Other nodes respond with their service information.
4. Heartbeats keep nodes active.
5. Offline nodes are cleaned after the configured timeout.

For the full protocol diagram, see the repository-level `docs/discovery.md` file.
