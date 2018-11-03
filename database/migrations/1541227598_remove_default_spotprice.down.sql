ALTER TABLE `amazon_eks_profile_node_pools` ALTER `spot_price` SET DEFAULT '0.2';

ALTER TABLE `amazon_ec2_profile_node_pools` ALTER `spot_price` SET DEFAULT '0.2';
