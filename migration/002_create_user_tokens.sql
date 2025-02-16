CREATE TABLE `user_tokens` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `token` text CHARACTER SET latin1 COLLATE latin1_swedish_ci,
  `status` int NOT NULL DEFAULT '0',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `user_tokens_users_id_fk` (`user_id`) USING BTREE,
  KEY `idx_id` (`id`) USING BTREE,
  CONSTRAINT `user_tokens_users_id_fk` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT
);

INSERT INTO user_tokens (id,user_id,token,status,created_at,updated_at) VALUES
	 (1,1,'YWRtaW5AbGlvbm5peC52bjpiMmViMWE5MS01ZWVlLTRkYTQtODFhMS1lMzRiNjBhZWE1MWY=',1,'2021-08-23 07:06:03','2021-08-23 07:06:03'),
	 (600,857,'c3VwcG9ydDJAbmV3dG9wZGVhbC5jb206OTUwZDdmYWEtMzY2ZC00YWIwLWI4ODktM2I4Y2Q2N2NhNDQ1',1,'2023-10-31 02:22:14','2023-10-31 02:22:14');


