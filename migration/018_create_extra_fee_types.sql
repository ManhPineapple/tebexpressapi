CREATE TABLE `extra_fee_types` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` int NOT NULL,
  `parent_id` bigint DEFAULT NULL,
  `is_refund` tinyint DEFAULT '0',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `fee` decimal(11,2) DEFAULT '0.00',
  `is_show` int DEFAULT '1',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_id` (`id`) USING BTREE
);

INSERT INTO extra_fee_types (id,name,status,parent_id,is_refund,created_at,updated_at,fee,is_show) VALUES
	 (1,'Phí covid',1,NULL,0,'2021-06-19 14:43:33','2021-06-19 14:43:39',0.00,1),
	 (2,'Phí quá cỡ',1,NULL,0,'2021-06-19 14:43:34','2021-06-19 14:43:39',0.00,1),
	 (3,'Phí sửa kích thước',1,NULL,0,'2021-06-19 14:43:34','2021-06-19 14:43:39',0.00,1),
	 (4,'Phí sửa trọng lượng',1,NULL,0,'2021-06-19 14:43:34','2021-06-19 14:43:39',0.00,1),
	 (6,'Phí sửa dịch vụ',1,NULL,0,'2021-06-19 14:43:34','2021-06-19 14:43:34',0.00,1),
	 (8,'Phí sửa đơn',1,NULL,0,'2021-06-19 14:43:34','2021-06-19 14:43:39',0.00,1),
	 (9,'Hoàn tiền',1,NULL,1,'2021-06-19 14:43:34','2021-06-19 14:43:39',0.00,1),
	 (10,'Phí phát sinh khác',1,NULL,0,'2021-07-28 16:45:21','2021-07-28 16:45:23',0.00,1),
	 (11,'Phí reship',1,NULL,0,'2022-03-01 04:13:57','2022-03-01 04:13:57',0.00,1),
	 (12,'Phụ phí cao điểm',1,NULL,0,'2022-05-13 03:29:05','2022-05-13 03:29:05',0.75,1);
INSERT INTO extra_fee_types (id,name,status,parent_id,is_refund,created_at,updated_at,fee,is_show) VALUES
	 (13,'Phí hủy label',1,NULL,0,'2022-08-10 13:51:09','2022-08-10 13:51:16',0.00,1),
	 (14,'Phí quá thể tích',1,NULL,0,'2022-10-03 02:04:12','2022-10-03 02:04:12',0.00,1),
	 (15,'Khuyến mãi theo cân nặng',1,NULL,0,'2022-10-03 02:04:12','2022-10-06 02:40:18',0.00,1),
	 (16,'Phí sửa địa chỉ',1,NULL,0,'2022-10-03 02:04:12','2022-10-06 02:40:18',0.00,1),
	 (17,'Phí Return',1,NULL,0,'2021-03-16 14:43:34','2021-03-16 14:43:34',0.00,1),
	 (18,'Phụ phí pin',1,NULL,0,'2021-06-20 14:43:34','2021-06-20 14:43:34',0.00,1),
	 (19,'Phí bảo hiểm',1,NULL,0,'2023-05-29 07:08:05','2023-05-29 07:08:05',0.00,1),
	 (20,'Coupon',1,NULL,0,'2023-07-18 11:12:54','2023-07-18 11:12:54',0.00,0),
	 (21,'Phí hoa hồng',1,NULL,0,'2023-07-18 11:12:54','2023-07-18 11:12:54',0.00,0),
	 (22,'Phụ phí kích thước',1,NULL,0,'2023-07-18 11:12:54','2023-07-18 11:12:54',0.00,1);
