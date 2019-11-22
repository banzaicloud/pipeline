ALTER TABLE "topology_kubeadms" ALTER COLUMN "extra_args" TYPE varchar(255) USING "extra_args"::text;
