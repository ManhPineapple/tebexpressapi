CREATE TABLE `settings` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `updated_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `user_id` bigint DEFAULT NULL,
  `key` varchar(100) CHARACTER SET latin1 COLLATE latin1_swedish_ci NOT NULL,
  `value` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` tinyint NOT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `settings_user_id_IDX` (`user_id`) USING BTREE
);