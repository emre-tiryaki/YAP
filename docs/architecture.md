# Project Architecture

```mermaid
flowchart LR
    n1["User"] --> n2
    subgraph Local
        n2["Frontend"] --> n3["Backend"]
        n3 --> n4["SQLite"]
    end
    n3 --> n5["Server"]
    n5 --> n6["PostgreSQL"]

    n1@{ icon: "fa:user", pos: "b"}
    n2@{ shape: text}
    n3@{ shape: text}
    n4@{ shape: disk}
    n5@{ shape: text}
    n6@{ shape: db}
```

Project is a personal and team project planner.

For personal use it will only use Frontend, Backend and SQLite.

## Frontend

Frontend will be written in React + TypeScript.
Bun is the package manager for it

## Backend

Backend will be written in go

It will use SQLite for local database

## Server

Will be written in go

Will use PostgreSQL for team management and synchronizing devices