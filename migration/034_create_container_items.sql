CREATE TABLE `container_items` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `container_id` bigint NOT NULL,
  `package_id` bigint NOT NULL,
  `status` int NOT NULL DEFAULT '0',
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `container_items_packages_id_fk` (`package_id`) USING BTREE,
  KEY `container_id` (`container_id`) USING BTREE,
  CONSTRAINT `container_items_ibfk_1` FOREIGN KEY (`container_id`) REFERENCES `containers` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `container_items_packages_id_fk` FOREIGN KEY (`package_id`) REFERENCES `packages` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);