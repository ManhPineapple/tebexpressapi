CREATE TABLE `package_refunds` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `package_id` bigint NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `status` int NOT NULL,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `package_id_idx` (`package_id`) USING BTREE,
  KEY `status_idx` (`status`) USING BTREE,
  CONSTRAINT `package_refunds_package_idfk` FOREIGN KEY (`package_id`) REFERENCES `packages` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);