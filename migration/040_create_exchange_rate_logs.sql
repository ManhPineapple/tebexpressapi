CREATE TABLE `exchange_rate_logs` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `rate` decimal(11,2) DEFAULT 0,
  `user_id` bigint NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE
);