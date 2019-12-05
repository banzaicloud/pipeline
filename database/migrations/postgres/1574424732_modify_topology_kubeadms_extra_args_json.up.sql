ALTER TABLE "topology_kubeadms" ALTER COLUMN "extra_args" TYPE json USING "extra_args"::json;
