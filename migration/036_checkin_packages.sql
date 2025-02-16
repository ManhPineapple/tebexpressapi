CREATE TABLE `checkin_packages` (
  `checkin_id` bigint NOT NULL,
  `package_id` bigint NOT NULL,
  `status` int NOT NULL DEFAULT '0',
  PRIMARY KEY (`checkin_id`,`package_id`) USING BTREE,
  KEY `package_idx` (`package_id`) USING BTREE,
  KEY `checkin_idx` (`checkin_id`) USING BTREE,
  CONSTRAINT `checkin_packages_package_id_fk` FOREIGN KEY (`package_id`) REFERENCES `packages` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `checkin_packags_chekin_id_fk` FOREIGN KEY (`checkin_id`) REFERENCES `checkin_requests` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);