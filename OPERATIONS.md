# Operations

Guia operativa minima para observar y operar `crm-agentic-flow` sin entrar directo a PostgreSQL.

## Servicios

Base de datos:

```bash
docker compose up -d db
```

API:

```bash
cd cmd/api
go run .
```

Worker:

```bash
cd cmd/event-processor
go run .
```

Smoke test automatizado del pipeline completo:

```bash
cd .
./scripts/smoke_test_pipeline.sh
```

Atajo con `make`:

```bash
cd .
make smoke
```

Flujo local similar a CI:

```bash
cd .
make ci
```

Notas:
- La API escucha en `http://localhost:8080`
- El `DATABASE_URL` por defecto apunta a `postgresql://postgres:postgres@localhost:5440/crm_agentic_flow`
- Redis ya no es parte del camino principal del backend

## Login

Seed de desarrollo:
- `email`: `admin@crmflow.local`
- `password`: `admin123`
- `tenant_slug`: `demo`

Obtener token:

```bash
curl -s http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@crmflow.local","password":"admin123","tenant_slug":"demo"}'
```

Export util:

```bash
export TOKEN="$(curl -s http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@crmflow.local","password":"admin123","tenant_slug":"demo"}' | jq -r .token)"
```

## Outbox

Listar eventos:

```bash
curl -s http://localhost:8080/outbox-events \
  -H "Authorization: Bearer $TOKEN"
```

Filtrar por estado y tipo:

```bash
curl -s 'http://localhost:8080/outbox-events?status=failed&event_type=organization.created&limit=10' \
  -H "Authorization: Bearer $TOKEN"
```

Ver un evento:

```bash
curl -s http://localhost:8080/outbox-events/1 \
  -H "Authorization: Bearer $TOKEN"
```

Resumen operativo:

```bash
curl -s http://localhost:8080/outbox-events/stats \
  -H "Authorization: Bearer $TOKEN"
```

Reencolar un evento:

```bash
curl -s -X POST http://localhost:8080/outbox-events/1/replay \
  -H "Authorization: Bearer $TOKEN"
```

Reencolar por filtro:

```bash
curl -s -X POST http://localhost:8080/outbox-events/replay \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"event_type":"organization.created","status":"processed","limit":25}'
```

## Webhooks

Crear endpoint:

```bash
curl -s -X POST http://localhost:8080/webhook-endpoints \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Primary Endpoint","target_url":"https://example.test/webhook","signing_secret":"super-secret"}'
```

Listar endpoints:

```bash
curl -s http://localhost:8080/webhook-endpoints \
  -H "Authorization: Bearer $TOKEN"
```

Actualizar endpoint:

```bash
curl -s -X PUT http://localhost:8080/webhook-endpoints/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Primary Endpoint Updated","target_url":"https://example.test/new-webhook","signing_secret":"super-secret-2","status":"active"}'
```

Borrar endpoint:

```bash
curl -s -X DELETE http://localhost:8080/webhook-endpoints/1 \
  -H "Authorization: Bearer $TOKEN" -i
```

Crear suscripcion:

```bash
curl -s -X POST http://localhost:8080/webhook-subscriptions \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"webhook_endpoint_id":1,"event_type":"organization.created"}'
```

Listar suscripciones:

```bash
curl -s http://localhost:8080/webhook-subscriptions \
  -H "Authorization: Bearer $TOKEN"
```

Actualizar suscripcion:

```bash
curl -s -X PUT http://localhost:8080/webhook-subscriptions/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"webhook_endpoint_id":1,"event_type":"organization.updated","is_active":false}'
```

Borrar suscripcion:

```bash
curl -s -X DELETE http://localhost:8080/webhook-subscriptions/1 \
  -H "Authorization: Bearer $TOKEN" -i
```

Listar deliveries:

```bash
curl -s 'http://localhost:8080/webhook-deliveries?status=failed&event_type=organization.created&webhook_endpoint_id=1' \
  -H "Authorization: Bearer $TOKEN"
```

Resumen operativo de deliveries:

```bash
curl -s http://localhost:8080/webhook-deliveries/stats \
  -H "Authorization: Bearer $TOKEN"
```

Reintentar un delivery:

```bash
curl -s -X POST http://localhost:8080/webhook-deliveries/1/replay \
  -H "Authorization: Bearer $TOKEN"
```

## Audit Log

Listar auditoria:

```bash
curl -s 'http://localhost:8080/audit-log?entity_type=organization&action=created&limit=20' \
  -H "Authorization: Bearer $TOKEN"
```

## Flujo de troubleshooting

1. Revisar `GET /outbox-events/stats` para ver si hay crecimiento en `failed` o `pending`.
2. Revisar `GET /outbox-events?status=failed` para inspeccionar payload y error.
3. Revisar `GET /webhook-deliveries/stats` para identificar si el problema esta concentrado en un endpoint.
4. Revisar `GET /webhook-deliveries?status=failed&webhook_endpoint_id=...`.
5. Corregir endpoint o suscripcion.
6. Reencolar con `POST /webhook-deliveries/{id}/replay` o `POST /outbox-events/replay`.

## Compatibilidad legacy

Los endpoints legacy `persons`, `flows` y `activities` siguen existiendo solo como compatibilidad temporal.

Para apagarlos:

```bash
export ENABLE_LEGACY_API=false
```

Con eso desaparecen del router principal.
