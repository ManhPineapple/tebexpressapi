CREATE TABLE `orders` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `count` int DEFAULT NULL,
  `status` tinyint NOT NULL,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `users_id_idx` (`user_id`) USING BTREE,
  KEY `status_idx` (`status`) USING BTREE,
  CONSTRAINT `orders_user_idfk` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);