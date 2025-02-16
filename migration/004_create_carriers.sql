CREATE TABLE `carriers` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `code` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` int DEFAULT NULL,
  `type` int DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `last_mile_carrier` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_id` (`id`) USING BTREE
);

INSERT INTO carriers (id,name,code,status,`type`,created_at,updated_at,last_mile_carrier) VALUES
	 (3,'UPS','UPS',1,2,'2021-06-30 07:24:51','2021-06-30 07:24:51','UPS'),
	 (5,'IB Blue','IBBLUE',1,1,'2021-06-30 07:24:51','2021-06-30 07:24:51','USPS');
