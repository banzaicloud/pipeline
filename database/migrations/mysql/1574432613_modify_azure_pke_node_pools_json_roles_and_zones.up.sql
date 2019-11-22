UPDATE `azure_pke_node_pools` SET `roles`='[]' WHERE `roles` = '';
UPDATE `azure_pke_node_pools` SET `roles`=CONCAT('["', REPLACE(`roles`, ',', '","'), '"]') WHERE `roles` <> '' AND NOT JSON_VALID(`roles`);
ALTER TABLE `azure_pke_node_pools` MODIFY `roles` json;

UPDATE `azure_pke_node_pools` SET `zones`='[]' WHERE `zones` = '';
UPDATE `azure_pke_node_pools` SET `zones`=CONCAT('["', REPLACE(`zones`, ',', '","'), '"]') WHERE `zones` <> '' AND NOT JSON_VALID(`zones`);
ALTER TABLE `azure_pke_node_pools` MODIFY `zones` json;
