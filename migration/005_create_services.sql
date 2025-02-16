CREATE TABLE `services` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `code` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `domestic_carrier_id` bigint DEFAULT NULL,
  `domestic_carrier_service` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ww_carrier_id` bigint DEFAULT NULL,
  `ww_carrier_service` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` tinyint DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `country` varchar(2) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `extra_fee_1` decimal(11,2) DEFAULT NULL,
  `extra_fee_2` decimal(11,2) DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_id` (`id`) USING BTREE,
  KEY `services_dcarrier_idfk` (`domestic_carrier_id`) USING BTREE,
  KEY `services_wcarrier_idfk` (`ww_carrier_id`) USING BTREE,
  CONSTRAINT `services_dcarrier_idfk` FOREIGN KEY (`domestic_carrier_id`) REFERENCES `carriers` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `services_wcarrier_idfk` FOREIGN KEY (`ww_carrier_id`) REFERENCES `carriers` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);

INSERT INTO services (id,name,code,domestic_carrier_id,domestic_carrier_service,ww_carrier_id,ww_carrier_service,status,created_at,updated_at,country,extra_fee_1,extra_fee_2) VALUES
	 (12,'Express','E',5,NULL,3,NULL,1,'2021-08-31 08:17:33','2021-08-31 08:17:33','US',NULL,NULL);
