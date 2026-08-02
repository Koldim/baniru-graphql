# GraphQL API

Go + Postgres. Всё через `main`, сателлиты напрямую не трогаем. Удаление через `deleted_at`.

Запуск:
```bash
cp .env.example .env
docker compose up --build
```

API: http://localhost:8080/graphql

Если схема БД уже битая: `docker compose down -v && docker compose up --build`

Примеры:

```graphql
mutation {
  createMain(input: { title: "Молоток", subObj: tools, description1: "ударный" }) {
    id title subId subObj description1
  }
}

query {
  main(id: "1") { id title subObj description1 type }
}

mutation {
  updateMain(id: "1", input: { title: "Молоток 2", subObj: tools, description1: "новый текст" }) {
    id title description1
  }
}

mutation { deleteMain(id: "1") }
```

`sub_obj` при update не меняется. После delete `main` вернёт null.
