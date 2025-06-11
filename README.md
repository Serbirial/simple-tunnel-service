# simple-tunnel-service
A super simple tunnel service, written in Go.

### Config files
## `server/secret.txt`
```
my_global_secret
```

## `server/service_secrets.json` (optional)
```json
{
  "ssh": "ssh_secret_456"
}
```
## `client/client.json`
```json
{
  "server_address": "your.server.ip:7000",
  "services": [
    {
      "service_name": "service1",
      "public_port": 9000,
      "local_address": "localhost:8080",
      "secret": "my_global_secret"
    },
    {
      "service_name": "ssh",
      "public_port": 9002,
      "local_address": "localhost:22",
      "secret": "ssh_secret_456"
    }
  ]
}
```

This will expose service1 (localhost:8080 on client) to 9000 on the server, and will forward ssh of the client to port `9002` on the server.

MAKE SURE YOU USE SECURE KEYS! **ANYONE** WITH THE KEY CAN MAP/HIJACK PORTS ON THE SERVER.