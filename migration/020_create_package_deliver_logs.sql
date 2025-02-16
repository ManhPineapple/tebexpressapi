CREATE TABLE `package_deliver_logs` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `package_id` bigint NOT NULL,
  `location` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `type` int DEFAULT NULL,
  `user_id` bigint DEFAULT NULL,
  `description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_id` (`id`) USING BTREE,
  KEY `package_deliver_logs_packages_id_fk` (`package_id`) USING BTREE,
  KEY `package_deliver_logs_users_id_fk` (`user_id`) USING BTREE,
  KEY `user_idx` (`user_id`) USING BTREE,
  CONSTRAINT `package_deliver_logs_packages_id_fk` FOREIGN KEY (`package_id`) REFERENCES `packages` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `package_deliver_logs_users_id_fk` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);