CREATE TABLE `warehouses` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` int NOT NULL DEFAULT '1',
  `type` int NOT NULL,
  `company` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `phone` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `address` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `city` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `state` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `zipcode` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `country` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `handling_fee` decimal(10,2) DEFAULT '0.00',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `time_open` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `time_active` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `link_address` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `manifest_active` int DEFAULT '1',
  PRIMARY KEY (`id`) USING BTREE,
  FULLTEXT KEY `warehouses_name_uindex` (`name`)
);

INSERT INTO warehouses (id,name,status,`type`,company,phone,address,city,state,zipcode,country,handling_fee,created_at,updated_at,time_open,time_active,link_address,manifest_active) VALUES
	 (1,'Dinh Thai Son',1,1,'Chicago','8155230042','5651 N Meade Ave','Chicago','IL','60646','US',0.00,'2021-10-19 09:02:58','2021-10-19 09:02:58',NULL,NULL,NULL,1),
	 (2,'Kho Hanoi',1,2,'ND','+84853312042','36 ngõ 1d trần quang diệu, đống đa, hn','Hanoi','VN','100000','VN',0.00,'2021-10-19 09:04:59','2021-10-19 09:04:59','Mở cửa lúc 9h','[{"date": "Từ thứ 2 đến thứ 7", "time": "09:00 AM - 5:30 PM"}]','https://goo.gl/maps/JZ9a48RSXQLt3nhS6',1),
	 (8,'Scott Ha',0,1,'ND','+61 416 984 420','3 Toohey Rd','Wetherill Park','NSW','2164','AU',0.00,'2022-11-24 10:57:32','2022-11-24 10:57:32',NULL,NULL,NULL,1);
