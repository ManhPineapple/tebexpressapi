CREATE TABLE `package_warehouse_costs` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `package_id` bigint DEFAULT NULL,
  `hub_id` bigint DEFAULT NULL,
  `carrier_id` bigint DEFAULT NULL,
  `cost` decimal(10,2) DEFAULT NULL,
  `status` int DEFAULT '1',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `org_cost` decimal(10,2) DEFAULT NULL,
  `zone` int DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `package_id_idx` (`package_id`) USING BTREE,
  KEY `pwc_hub_idfk` (`hub_id`) USING BTREE,
  KEY `pwc_carrier_idfk` (`carrier_id`) USING BTREE,
  CONSTRAINT `pwc_carrier_idfk` FOREIGN KEY (`carrier_id`) REFERENCES `carriers` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `pwc_hub_idfk` FOREIGN KEY (`hub_id`) REFERENCES `warehouses` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `pwc_package_idfk` FOREIGN KEY (`package_id`) REFERENCES `packages` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);