CREATE TABLE `users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `username` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `password` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `email` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `phone_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `full_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `birthday` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` int NOT NULL,
  `role` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `package` int NOT NULL DEFAULT '0',
  `balance` decimal(10,2) DEFAULT NULL,
  `rewards` decimal(10,2) DEFAULT NULL,
  `class` int NOT NULL DEFAULT '1',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `warehouse_id` bigint DEFAULT NULL,
  `slack_id` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `referral_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ref_id` bigint DEFAULT NULL,
  `point` int DEFAULT '0',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `users_email_uindex` (`email`) USING BTREE,
  UNIQUE KEY `users_phone_number_uindex` (`phone_number`) USING BTREE
);

INSERT INTO users (id,username,password,email,phone_number,full_name,birthday,status,`role`,package,balance,rewards,class,created_at,updated_at,warehouse_id,slack_id,referral_code,ref_id,`point`) VALUES
	 (1,'','$2a$04$BTdw43CKv12h0DInXN8ahe9JJU2LEmEiuBQ28aHrNbyjlg37CBcHW','admin@ananbay.com','8888888888','Ananbay Admin','',1,'admin',0,0.00,0.00,1,'2021-08-23 07:06:03','2023-07-05 10:43:35',NULL,NULL,'WTg5NPU8UfOsTKUEzhCQ',NULL,0),
   (857,'','$2a$04$BTdw43CKv12h0DInXN8ahe9JJU2LEmEiuBQ28aHrNbyjlg37CBcHW','support2@newtopdeal.com','0828500046','ntdlionbay','',1,'customer',2,-102.36,0.00,2,'2023-10-31 02:22:14','2024-02-18 17:20:13',NULL,NULL,'',NULL,0);
