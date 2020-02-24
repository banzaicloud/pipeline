CREATE TABLE "public"."processes" (
    "id" text NOT NULL,
    "parent_id" text,
    "org_id" int4 NOT NULL,
    "type" text NOT NULL,
    "log" text,
    "resource_id" text NOT NULL,
    "status" text NOT NULL,
    "started_at" timestamptz NOT NULL DEFAULT now(),
    "finished_at" timestamptz,
    PRIMARY KEY ("id")
);

CREATE INDEX idx_processes_parent_id ON processes USING btree (parent_id);

CREATE INDEX idx_start_time_end_time ON processes USING btree (started_at, finished_at);

CREATE TABLE "public"."process_events" (
    "id" serial,
    "process_id" text NOT NULL,
    "type" text NOT NULL,
    "log" text,
    "status" text NOT NULL,
    "timestamp" timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT "process_events_process_id_processes_id_foreign" FOREIGN KEY ("process_id") REFERENCES "public"."processes"("id") ON DELETE RESTRICT ON UPDATE RESTRICT
);

CREATE UNIQUE INDEX process_events_pkey ON process_events USING btree (id);

ALTER TABLE "process_events" add constraint "process_events_pkey" PRIMARY KEY using index "process_events_pkey";
