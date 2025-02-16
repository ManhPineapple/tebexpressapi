CREATE TABLE `check_price_logs` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `data` json DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE
);