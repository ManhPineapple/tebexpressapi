CREATE TABLE `user_infos` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `debt_max_amount` decimal(20,2) DEFAULT '0.00',
  `debt_max_day` int DEFAULT '0',
  `debt_time` datetime DEFAULT NULL,
  `appraiser_id` bigint DEFAULT NULL,
  `tax_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `volume` varchar(255) CHARACTER SET latin1 COLLATE latin1_swedish_ci DEFAULT NULL,
  `item_type` varchar(255) CHARACTER SET latin1 COLLATE latin1_swedish_ci DEFAULT NULL,
  `warehouse_address` varchar(255) CHARACTER SET latin1 COLLATE latin1_swedish_ci DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `cancel_max_amount` decimal(10,2) DEFAULT '200.00',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `user_id` (`user_id`) USING BTREE,
  CONSTRAINT `user_infos_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);

INSERT INTO user_infos (id,user_id,debt_max_amount,debt_max_day,debt_time,appraiser_id,tax_code,volume,item_type,warehouse_address,updated_at,cancel_max_amount) VALUES
	 (421,857,15000.00,180,'2024-05-02 07:23:47',0,'','','','','2024-01-05 05:19:17',300.00);

ALTER TABLE user_infos
ADD COLUMN refund_day INT DEFAULT 14;