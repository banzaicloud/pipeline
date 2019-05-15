CREATE UNIQUE INDEX idx_azure_pke_np_cluster_id_name ON `azure_pke_node_pools`(cluster_id, `name`);
CREATE UNIQUE INDEX idx_azure_pke_cluster_id ON `azure_pke_clusters`(cluster_id);
