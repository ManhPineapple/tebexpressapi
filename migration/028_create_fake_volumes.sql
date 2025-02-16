CREATE TABLE `fake_volumes` (
  `id` int NOT NULL AUTO_INCREMENT,
  `milestone` float DEFAULT '0',
  `weight` float DEFAULT '0',
  `dimension` float DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `service_id` (`milestone`) USING BTREE
);