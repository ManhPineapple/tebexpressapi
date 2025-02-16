CREATE TABLE `states` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) CHARACTER SET latin1 COLLATE latin1_swedish_ci NOT NULL,
  `country` varchar(255) CHARACTER SET latin1 COLLATE latin1_swedish_ci NOT NULL,
  `code` varchar(255) CHARACTER SET latin1 COLLATE latin1_swedish_ci NOT NULL,
  `status` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE
);

INSERT INTO states (name,country,code,status) VALUES
	 ('Alabama','US','AL',1),
	 ('Alaska','US','AK',2),
	 ('Arizona','US','AZ',1),
	 ('Arkansas','US','AR',1),
	 ('California','US','CA',1),
	 ('Colorado','US','CO',1),
	 ('Connecticut','US','CT',1),
	 ('Delaware','US','DE',1),
	 ('District Of Columbia','US','DC',1),
	 ('Florida','US','FL',1);
INSERT INTO states (name,country,code,status) VALUES
	 ('Georgia','US','GA',1),
	 ('Idaho','US','ID',1),
	 ('Illinois','US','IL',1),
	 ('Indiana','US','IN',1),
	 ('Iowa','US','IA',1),
	 ('Kansas','US','KS',1),
	 ('Kentucky','US','KY',1),
	 ('Louisiana','US','LA',1),
	 ('Maine','US','ME',1),
	 ('Maryland','US','MD',1);
INSERT INTO states (name,country,code,status) VALUES
	 ('Massachusetts','US','MA',1),
	 ('Michigan','US','MI',1),
	 ('Minnesota','US','MN',1),
	 ('Mississippi','US','MS',1),
	 ('Missouri','US','MO',1),
	 ('Montana','US','MT',1),
	 ('Nebraska','US','NE',1),
	 ('Nevada','US','NV',1),
	 ('New Hampshire','US','NH',1),
	 ('New Jersey','US','NJ',1);
INSERT INTO states (name,country,code,status) VALUES
	 ('New Mexico','US','NM',1),
	 ('New York','US','NY',1),
	 ('North Carolina','US','NC',1),
	 ('North Dakota','US','ND',1),
	 ('Ohio','US','OH',1),
	 ('Oklahoma','US','OK',1),
	 ('Oregon','US','OR',1),
	 ('Pennsylvania','US','PA',1),
	 ('Rhode Island','US','RI',1),
	 ('South Carolina','US','SC',1);
INSERT INTO states (name,country,code,status) VALUES
	 ('South Dakota','US','SD',1),
	 ('Tennessee','US','TN',1),
	 ('Texas','US','TX',1),
	 ('Utah','US','UT',1),
	 ('Vermont','US','VT',1),
	 ('Virginia','US','VA',1),
	 ('Washington','US','WA',1),
	 ('West Virginia','US','WV',1),
	 ('Wisconsin','US','WI',1),
	 ('Wyoming','US','WY',1);
INSERT INTO states (name,country,code,status) VALUES
	 ('Armed Forces (AA)','US','AA',2),
	 ('Armed Forces Pacific','US','AP',2),
	 ('Armed Forces Europe','US','AE',2),
	 ('Puerto Rico','US','PR',2),
	 ('Virgin Islands','US','VI',2),
	 ('Hawaii','US','HI',2),
	 ('Guam','US','GU',2),
	 ('Northern Mariana Islands','US','MP',2),
	 ('New South Wales','AU','NSW',1),
	 ('Queensland','AU','QLD',1);
INSERT INTO states (name,country,code,status) VALUES
	 ('South Australia','AU','SA',1),
	 ('Tasmania','AU','TAS',1),
	 ('Victoria','AU','VIC',1),
	 ('Western Australia','AU','WA',1),
	 ('Australian Capital Territory','AU','ACT',1),
	 ('Northern Territory','AU','NT',1);
