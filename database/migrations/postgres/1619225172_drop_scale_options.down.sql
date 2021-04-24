CREATE TABLE "scale_options" (
     "id" serial,
     "cluster_id" integer,
     "enabled" boolean,
     "desired_cpu" numeric,
     "desired_mem" numeric,
     "desired_gpu" integer,
     "on_demand_pct" integer,
     "excludes" text,
     "keep_desired_capacity" boolean,
     PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_scale_options_cluster_id ON "scale_options"(cluster_id);
