-- инициализация 

CREATE TYPE chair_type AS ENUM ('abc', 'cde');

CREATE TYPE sub_obj AS ENUM ('tools', 'tables', 'chairs');

CREATE TABLE main (
    id          SERIAL PRIMARY KEY,
    title       TEXT NOT NULL,
    sub_id      INTEGER,
    sub_obj     sub_obj NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    update_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE TABLE tools (
    id            SERIAL PRIMARY KEY,
    main_id       INTEGER REFERENCES main(id),
    description1  TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    update_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE TABLE "tables" (
    id            SERIAL PRIMARY KEY,
    main_id       INTEGER REFERENCES main(id),
    description2  TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    update_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE TABLE chairs (
    id            SERIAL PRIMARY KEY,
    main_id       INTEGER REFERENCES main(id),
    description3  TEXT NOT NULL DEFAULT '',
    type          chair_type NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    update_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_main_deleted_at ON main (deleted_at);
CREATE INDEX idx_tools_main_id ON tools (main_id);
CREATE INDEX idx_tables_main_id ON "tables" (main_id);
CREATE INDEX idx_chairs_main_id ON chairs (main_id);