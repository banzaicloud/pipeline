ALTER TABLE google_gke_node_pools ADD COLUMN `preemptible` tinyint(1) DEFAULT '0';
ALTER TABLE google_gke_profile_node_pools ADD COLUMN `preemptible` tinyint(1) DEFAULT '0';
