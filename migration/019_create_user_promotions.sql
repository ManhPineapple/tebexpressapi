CREATE TABLE `user_promotions` (
  `user_id` bigint NOT NULL,
  `promotion_id` bigint NOT NULL,
  `status` int NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`user_id`,`promotion_id`) USING BTREE,
  KEY `up_promotion_idfk` (`promotion_id`) USING BTREE,
  CONSTRAINT `up_promotion_idfk` FOREIGN KEY (`promotion_id`) REFERENCES `promotions` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `up_user_idfk` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);