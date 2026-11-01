# Tasker


A small task manager, written in Go.

The whole backend (the HTTP server, the PostgreSQL store and the task logic) is written by hand, without AI. These days it's easy to let AI write everything, and I noticed I was losing the habit of writing code myself. So this project is my way of getting back into it: at least the backend is mine, line by line. The commit history shows the real process, from an in-memory map to a proper database.

The tests and the frontend (a GitHub Projects style board in `internal/server/static/`) were made with AI. I'm not writing tests by hand, AI is great for that.

<img src="https://images.squarespace-cdn.com/content/v1/64dce4d346f8190ef5f2d84f/721ddc70-1e19-43cb-9a22-1263f54cb624/giphy.gif?format=750w" alt="Writing code by hand" width="300">

## What it does

Tasks have a title, a description and one of three statuses: Todo, In Progress or Done. You can drag them between columns on the board, and every move is saved to the database.

![Board](assets/board.png)

Click a card to see the details, change the status or delete it:

![Task details](assets/task-details.png)

## Structure

```
cmd/tasker/            entry point
internal/server/       HTTP handlers (net/http only, no frameworks)
internal/database/     PostgreSQL task store
internal/task/         task model and errors
migrations/            SQL schema
```

## API

| Method   | Path          | What it does                                  |
| -------- | ------------- | --------------------------------------------- |
| `GET`    | `/health`     | health check                                  |
| `GET`    | `/tasks`      | list all tasks                                |
| `GET`    | `/tasks/{id}` | get one task                                  |
| `POST`   | `/tasks`      | create a task: `{"title": "...", "description": "..."}` |
| `PATCH`  | `/tasks/{id}` | change status: `{"status": 0 \| 1 \| 2}`      |
| `DELETE` | `/tasks/{id}` | delete a task                                 |

## Running

Start PostgreSQL and create the table:

```
docker compose up -d
docker compose exec -T postgres psql -U tasker -d tasker < migrations/001_create_tasks.sql
```

Run the server:

```
DATABASE_URL=postgres://tasker:tasker_dev@127.0.0.1:5432/tasker go run ./cmd/tasker
```

Then open http://localhost:5001.

Tests:

```
go test ./...
```
