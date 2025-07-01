CREATE TABLE tasks (
    uuid        uuid        PRIMARY KEY,
    version     int         NOT NULL,
    task_data   jsonb       NOT NULL
);
