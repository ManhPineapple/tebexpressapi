CREATE TABLE `container_boxes` (
  `id` int NOT NULL AUTO_INCREMENT,
  `length` double DEFAULT NULL,
  `width` double DEFAULT NULL,
  `height` double DEFAULT NULL,
  `max_weight` double DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE
);

INSERT INTO container_boxes (id,`length`,width,height,max_weight,created_at,updated_at) VALUES
	 (2,60.0,40.0,40.0,32.0,'2021-08-10 10:18:07','2021-08-10 10:18:07');
