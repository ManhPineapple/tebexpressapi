CREATE TABLE `promotions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(50) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci NOT NULL,
  `status` int NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `description` text CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci,
  `type` int NOT NULL DEFAULT '1',
  `user_id` bigint DEFAULT NULL,
  `s3_path_price` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL,
  `s3_path_weight` varchar(255) CHARACTER SET utf8mb3 COLLATE utf8mb3_general_ci DEFAULT NULL,
  `price` float DEFAULT '0',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `promotions_user_idfk` (`user_id`) USING BTREE,
  CONSTRAINT `promotions_user_idfk` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);