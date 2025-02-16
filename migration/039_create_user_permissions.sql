CREATE TABLE `user_permissions` (
  `support_id` bigint NOT NULL,
  `customer_id` bigint NOT NULL,
  PRIMARY KEY (`support_id`, `customer_id`) USING BTREE,
  CONSTRAINT `user_permissions_support_id_fk` FOREIGN KEY (`support_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  CONSTRAINT `user_permissions_customer_id_fk` FOREIGN KEY (`customer_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);