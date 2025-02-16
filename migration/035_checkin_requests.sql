CREATE TABLE `checkin_requests` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `status` int NOT NULL,
  `close_user_id` bigint DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `close_user_id` (`close_user_id`) USING BTREE,
  CONSTRAINT `close_user_id` FOREIGN KEY (`close_user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);