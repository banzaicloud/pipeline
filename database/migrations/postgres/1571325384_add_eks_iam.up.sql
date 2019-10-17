ALTER TABLE "amazon_eks_clusters"
ADD COLUMN "default_user" BOOLEAN,
ADD COLUMN "cluster_role_id" TEXT,
ADD COLUMN "node_instance_role_id" TEXT;
