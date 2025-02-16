CREATE TABLE `fake_dimensions` (
  `id` int NOT NULL AUTO_INCREMENT,
  `from` float DEFAULT '0',
  `to` float DEFAULT '0',
  `rate` float DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `service_id` (`from`) USING BTREE
);