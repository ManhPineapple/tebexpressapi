CREATE TABLE `bills` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `code` varchar(50) CHARACTER SET latin1 COLLATE latin1_swedish_ci DEFAULT NULL,
  `user_id` bigint NOT NULL,
  `shipping_fee` decimal(11,2) DEFAULT NULL,
  `extra_fee` decimal(11,2) DEFAULT NULL,
  `status` int NOT NULL DEFAULT '0',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `bills_code_uidx` (`code`) USING BTREE,
  KEY `bills_users_id_fk` (`user_id`) USING BTREE,
  KEY `idx_id` (`id`) USING BTREE,
  CONSTRAINT `bills_users_id_fk` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);