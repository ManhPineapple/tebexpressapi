CREATE TABLE `manifests` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `shipment_id` bigint DEFAULT NULL,
  `manifest_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `manifest_url` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `package_id` bigint DEFAULT NULL,
  `container_id` bigint DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `manifests_shipments_id_fk` (`shipment_id`) USING BTREE,
  KEY `manifests_containers_id_fk` (`container_id`) USING BTREE,
  KEY `manifests_packages_id_fk` (`package_id`) USING BTREE,
  CONSTRAINT `manifests_containers_id_fk` FOREIGN KEY (`container_id`) REFERENCES `containers` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `manifests_packages_id_fk` FOREIGN KEY (`package_id`) REFERENCES `packages` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `manifests_shipments_id_fk` FOREIGN KEY (`shipment_id`) REFERENCES `shipments` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);