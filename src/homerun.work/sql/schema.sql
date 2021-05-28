-- MySQL dump 10.13  Distrib 8.0.19, for Win64 (x86_64)
--
-- Host: 127.0.0.1    Database: homerundb_dev
-- ------------------------------------------------------
-- Server version	8.0.22

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `campaign`
--

DROP TABLE IF EXISTS `campaign`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `campaign` (
  `id` binary(16) NOT NULL,
  `user_id` binary(16) NOT NULL,
  `provider_id` binary(16) NOT NULL,
  `external_id` binary(16) NOT NULL,
  `data` json DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.campaign.provider_id` (`provider_id`),
  KEY `idx.campaign.user_id` (`user_id`),
  KEY `idx.campaign.external_id` (`external_id`),
  CONSTRAINT `fk.campaign.provider_id` FOREIGN KEY (`provider_id`) REFERENCES `provider` (`id`),
  CONSTRAINT `fk.campaign.user_id` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `client`
--

DROP TABLE IF EXISTS `client`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `client` (
  `id` binary(16) NOT NULL,
  `provider_id` binary(16) NOT NULL,
  `user_id` binary(16) DEFAULT NULL,
  `email` varchar(100) NOT NULL,
  `invited` datetime DEFAULT NULL,
  `disable_emails` bit(1) NOT NULL DEFAULT b'0',
  `data` json DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.client.user_id` (`user_id`),
  KEY `idx.client.email` (`email`),
  KEY `idx.client.provider_id` (`provider_id`),
  CONSTRAINT `fk.client.provider_id` FOREIGN KEY (`provider_id`) REFERENCES `provider` (`id`),
  CONSTRAINT `fk.client.user_id` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `content`
--

DROP TABLE IF EXISTS `content`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `content` (
  `id` binary(16) NOT NULL,
  `type` tinyint NOT NULL,
  `data` json DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `coupon`
--

DROP TABLE IF EXISTS `coupon`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `coupon` (
  `id` binary(16) NOT NULL,
  `provider_id` binary(16) NOT NULL,
  `code` varchar(10) NOT NULL,
  `start` datetime NOT NULL,
  `end` datetime NOT NULL,
  `data` json DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.coupon.provider_id` (`provider_id`) /*!80000 INVISIBLE */,
  KEY `idx.coupon.code` (`code`),
  CONSTRAINT `fk.coupon.provider_id` FOREIGN KEY (`provider_id`) REFERENCES `provider` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `email_verify_token`
--

DROP TABLE IF EXISTS `email_verify_token`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `email_verify_token` (
  `user_id` binary(16) NOT NULL,
  `token` varchar(64) NOT NULL,
  `expiration` bigint NOT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`token`),
  KEY `idx.email_verify_token.user_id` (`user_id`) /*!80000 INVISIBLE */,
  KEY `idx.email_verify_token.token` (`token`),
  CONSTRAINT `fk.email_verify_token.user_id` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `faq`
--

DROP TABLE IF EXISTS `faq`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `faq` (
  `id` binary(16) NOT NULL,
  `provider_id` binary(16) NOT NULL,
  `idx` smallint NOT NULL DEFAULT '16384',
  `data` json DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.qa.provider_id` (`provider_id`),
  CONSTRAINT `fk.qa.provider_id` FOREIGN KEY (`provider_id`) REFERENCES `provider` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `img`
--

DROP TABLE IF EXISTS `img`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `img` (
  `id` binary(16) NOT NULL,
  `user_id` binary(16) NOT NULL,
  `provider_id` binary(16) NOT NULL,
  `secondary_id` binary(16) NOT NULL,
  `type` tinyint NOT NULL,
  `path` varchar(100) NOT NULL,
  `file_src` varchar(100) NOT NULL,
  `file_resized` varchar(100) DEFAULT NULL,
  `idx` smallint NOT NULL DEFAULT '0',
  `data` json DEFAULT NULL,
  `processing` bit(1) NOT NULL DEFAULT b'0',
  `processing_time` datetime DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.img.user_id` (`user_id`),
  KEY `idx.img.secondary_id` (`secondary_id`),
  KEY `idx.img.provider_id` (`provider_id`),
  CONSTRAINT `fk.img.provider_id` FOREIGN KEY (`provider_id`) REFERENCES `provider` (`id`),
  CONSTRAINT `fk.img.user_id` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `message`
--

DROP TABLE IF EXISTS `message`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `message` (
  `id` binary(16) NOT NULL,
  `secondary_id` binary(16) NOT NULL,
  `from_user_id` binary(16) DEFAULT NULL,
  `from_client_id` binary(16) DEFAULT NULL,
  `to_user_id` binary(16) DEFAULT NULL,
  `to_client_id` binary(16) DEFAULT NULL,
  `to_email` varchar(100) NOT NULL,
  `email_processing` bit(1) NOT NULL DEFAULT b'0',
  `email_processing_time` datetime DEFAULT NULL,
  `email_processed` bit(1) NOT NULL DEFAULT b'0',
  `sms_processing` bit(1) NOT NULL DEFAULT b'0',
  `sms_processing_time` datetime DEFAULT NULL,
  `sms_processed` bit(1) NOT NULL DEFAULT b'0',
  `data` json DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.message.to_user_id` (`to_user_id`) /*!80000 INVISIBLE */,
  KEY `idx.message.to_client_id` (`to_client_id`),
  KEY `idx.message.from_user_id` (`from_user_id`),
  KEY `idx.message.from_client_id` (`from_client_id`),
  KEY `idx.message.to_email` (`to_email`),
  KEY `idx.message.secondary_id` (`secondary_id`),
  CONSTRAINT `fk.message.from_client_id` FOREIGN KEY (`from_client_id`) REFERENCES `client` (`id`),
  CONSTRAINT `fk.message.from_user_id` FOREIGN KEY (`from_user_id`) REFERENCES `user` (`id`),
  CONSTRAINT `fk.message.to_client_id` FOREIGN KEY (`to_client_id`) REFERENCES `client` (`id`),
  CONSTRAINT `fk.message.to_user_id` FOREIGN KEY (`to_user_id`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `notification`
--

DROP TABLE IF EXISTS `notification`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `notification` (
  `id` binary(16) NOT NULL,
  `user_id` binary(16) NOT NULL,
  `secondary_id` binary(16) NOT NULL,
  `type` smallint NOT NULL,
  `external_id` varchar(100) DEFAULT NULL,
  `send_date` datetime NOT NULL,
  `data` json DEFAULT NULL,
  `processing` bit(1) NOT NULL DEFAULT b'0',
  `processing_time` datetime DEFAULT NULL,
  `processed` bit(1) NOT NULL DEFAULT b'0',
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.notification.user_id` (`user_id`) /*!80000 INVISIBLE */,
  KEY `idx.notification.secondary_id` (`secondary_id`),
  CONSTRAINT `fk.notification.user_id` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `payment`
--

DROP TABLE IF EXISTS `payment`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `payment` (
  `id` binary(16) NOT NULL,
  `provider_id` binary(16) NOT NULL,
  `secondary_id` binary(16) NOT NULL,
  `friendly_id` varchar(10) NOT NULL,
  `type` smallint NOT NULL,
  `amount` int NOT NULL,
  `invoiced` datetime DEFAULT NULL,
  `paid` datetime DEFAULT NULL,
  `captured` datetime DEFAULT NULL,
  `stripe_id` varchar(100) DEFAULT NULL,
  `stripe_session_id` varchar(100) DEFAULT NULL,
  `stripe_account_id` varchar(100) DEFAULT NULL,
  `stripe_data` json DEFAULT NULL,
  `paypal_id` varchar(100) DEFAULT NULL,
  `paypal_data` json DEFAULT NULL,
  `data` json DEFAULT NULL,
  `external_data` json DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq.payment.friendly_id` (`friendly_id`),
  KEY `idx.payment.provider_id` (`provider_id`) /*!80000 INVISIBLE */,
  KEY `idx.payment.secondary_id` (`secondary_id`) /*!80000 INVISIBLE */,
  KEY `idx.payment.stripe_id` (`stripe_id`),
  KEY `idx.payment.paypal_id` (`paypal_id`),
  CONSTRAINT `fk.payment.provider_id` FOREIGN KEY (`provider_id`) REFERENCES `provider` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `paypal_event`
--

DROP TABLE IF EXISTS `paypal_event`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `paypal_event` (
  `id` varchar(50) NOT NULL,
  `data` json DEFAULT NULL,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `provider`
--

DROP TABLE IF EXISTS `provider`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `provider` (
  `id` binary(16) NOT NULL,
  `user_id` binary(16) NOT NULL,
  `url_name` varchar(100) NOT NULL,
  `url_name_friendly` varchar(100) DEFAULT NULL,
  `domain` varchar(100) DEFAULT NULL,
  `calendar_google_id` varchar(100) DEFAULT NULL,
  `calendar_google_update` bit(1) NOT NULL DEFAULT b'0',
  `calendar_google_processing` bit(1) NOT NULL DEFAULT b'0',
  `calendar_google_processing_time` datetime DEFAULT NULL,
  `calendar_google_data` json DEFAULT NULL,
  `data` json DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.provider.user_id` (`user_id`),
  KEY `idx.provider.url_name` (`url_name`) /*!80000 INVISIBLE */,
  KEY `idx.provider.url_name_friendly` (`url_name_friendly`),
  KEY `idx.provider.created` (`created`,`id`),
  KEY `idx.provider.domain` (`domain`),
  CONSTRAINT `fk.provider.user_id` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `provider_user`
--

DROP TABLE IF EXISTS `provider_user`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `provider_user` (
  `id` binary(16) NOT NULL,
  `provider_id` binary(16) NOT NULL,
  `login` varchar(100) NOT NULL,
  `user_id` binary(16) DEFAULT NULL,
  `data` json DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.provider_user.provider_id` (`provider_id`) /*!80000 INVISIBLE */,
  KEY `idx.provider_user.user_id` (`user_id`),
  KEY `idx.provider_user.login` (`login`),
  CONSTRAINT `fk.provider_user.provider_id` FOREIGN KEY (`provider_id`) REFERENCES `provider` (`id`),
  CONSTRAINT `fk.provider_user.user_id` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `pwd_reset_token`
--

DROP TABLE IF EXISTS `pwd_reset_token`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `pwd_reset_token` (
  `user_id` binary(16) NOT NULL,
  `token` varchar(64) NOT NULL,
  `expiration` bigint NOT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`token`),
  KEY `idx.pwd_reset_token.user_id` (`user_id`) /*!80000 INVISIBLE */,
  KEY `idx.pwd_reset_token.token` (`token`),
  CONSTRAINT `fk.pwd_reset_token.user_id` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `service`
--

DROP TABLE IF EXISTS `service`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `service` (
  `id` binary(16) NOT NULL,
  `provider_id` binary(16) NOT NULL,
  `type` smallint NOT NULL,
  `idx` smallint NOT NULL DEFAULT '16384',
  `data` json DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.service.provider_id` (`provider_id`),
  KEY `idx.service.type` (`type`),
  CONSTRAINT `fk.service.provider_id` FOREIGN KEY (`provider_id`) REFERENCES `provider` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `service_booking`
--

DROP TABLE IF EXISTS `service_booking`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `service_booking` (
  `id` binary(16) NOT NULL,
  `provider_id` binary(16) NOT NULL,
  `provider_user_id` binary(16) DEFAULT NULL,
  `service_type` smallint NOT NULL,
  `service_id` binary(16) NOT NULL,
  `client_id` binary(16) NOT NULL,
  `parent_id` binary(16) DEFAULT NULL,
  `time_start` datetime NOT NULL,
  `time_end` datetime NOT NULL,
  `time_start_padded` datetime NOT NULL,
  `time_end_padded` datetime NOT NULL,
  `viewed` bit(1) NOT NULL DEFAULT b'0',
  `confirmed` bit(1) DEFAULT b'0',
  `client_created` bit(1) NOT NULL DEFAULT b'0',
  `recurrence_start` datetime DEFAULT NULL,
  `recurrence_rules` varchar(100) DEFAULT NULL,
  `recurrence_instance_end` datetime DEFAULT NULL,
  `recurrence_processing` bit(1) NOT NULL DEFAULT b'0',
  `recurrence_processing_time` datetime DEFAULT NULL,
  `event_google_id` varchar(100) DEFAULT NULL,
  `event_google_update` bit(1) NOT NULL DEFAULT b'0',
  `event_google_delete` bit(1) NOT NULL DEFAULT b'0',
  `event_google_processing` bit(1) NOT NULL DEFAULT b'0',
  `event_google_processing_time` datetime DEFAULT NULL,
  `event_google_data` json DEFAULT NULL,
  `meeting_zoom_id` varchar(100) DEFAULT NULL,
  `meeting_zoom_update` bit(1) NOT NULL DEFAULT b'0',
  `meeting_zoom_delete` bit(1) NOT NULL DEFAULT b'0',
  `meeting_zoom_processing` bit(1) NOT NULL DEFAULT b'0',
  `meeting_zoom_processing_time` datetime DEFAULT NULL,
  `meeting_zoom_data` json DEFAULT NULL,
  `data` json DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.service_booking.service_id` (`service_id`),
  KEY `idx.service_booking.client_id` (`client_id`),
  KEY `idx.service_booking.service_type` (`service_type`),
  KEY `idx.service_booking.parent_id` (`parent_id`),
  KEY `idx.service_booking.provider_id` (`provider_id`),
  KEY `idx.service_booking.provider_user_id` (`provider_user_id`),
  CONSTRAINT `fk.service_booking.client_id` FOREIGN KEY (`client_id`) REFERENCES `client` (`id`),
  CONSTRAINT `fk.service_booking.provider_id` FOREIGN KEY (`provider_id`) REFERENCES `provider` (`id`),
  CONSTRAINT `fk.service_booking.provider_user_id` FOREIGN KEY (`provider_user_id`) REFERENCES `provider_user` (`id`),
  CONSTRAINT `fk.service_booking.service_id` FOREIGN KEY (`service_id`) REFERENCES `service` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `service_provider_user`
--

DROP TABLE IF EXISTS `service_provider_user`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `service_provider_user` (
  `id` binary(16) NOT NULL,
  `provider_id` binary(16) NOT NULL,
  `service_id` binary(16) NOT NULL,
  `provider_user_id` binary(16) NOT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.service_provider_user.service_id` (`service_id`) /*!80000 INVISIBLE */,
  KEY `idx.service_provider_user.provider_user_id` (`provider_user_id`),
  KEY `idx.service_provider_user.provider_id` (`provider_id`),
  CONSTRAINT `fk.service_provider_user.provider_id` FOREIGN KEY (`provider_id`) REFERENCES `provider` (`id`),
  CONSTRAINT `fk.service_provider_user.provider_user_id` FOREIGN KEY (`provider_user_id`) REFERENCES `provider_user` (`id`),
  CONSTRAINT `fk.service_provider_user.service_id` FOREIGN KEY (`service_id`) REFERENCES `service` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `stripe_event`
--

DROP TABLE IF EXISTS `stripe_event`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `stripe_event` (
  `id` varchar(50) NOT NULL,
  `data` json DEFAULT NULL,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `testimonial`
--

DROP TABLE IF EXISTS `testimonial`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `testimonial` (
  `id` binary(16) NOT NULL,
  `provider_id` binary(16) NOT NULL,
  `data` json DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.testimonial.provider_id` (`provider_id`),
  CONSTRAINT `fk.testimonial.provider_id` FOREIGN KEY (`provider_id`) REFERENCES `provider` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `url_short`
--

DROP TABLE IF EXISTS `url_short`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `url_short` (
  `id` binary(16) NOT NULL,
  `url_short` varchar(100) NOT NULL,
  `url` varchar(200) NOT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.url_short.url_short` (`url_short`) /*!80000 INVISIBLE */,
  KEY `uq.url_short.url` (`url`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user`
--

DROP TABLE IF EXISTS `user`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user` (
  `id` binary(16) NOT NULL,
  `login` varchar(100) NOT NULL,
  `email` varchar(100) NOT NULL,
  `password` binary(60) DEFAULT NULL,
  `is_oauth` bit(1) NOT NULL DEFAULT b'0',
  `email_verified` bit(1) NOT NULL DEFAULT b'0',
  `disable_emails` bit(1) NOT NULL DEFAULT b'0',
  `last_login` datetime DEFAULT NULL,
  `token_zoom_data` json DEFAULT NULL,
  `data` json DEFAULT NULL,
  `deleted` bit(1) NOT NULL DEFAULT b'0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx.user.email` (`email`),
  KEY `idx.user.login` (`login`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `zoom_event`
--

DROP TABLE IF EXISTS `zoom_event`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `zoom_event` (
  `id` varchar(50) NOT NULL,
  `data` json DEFAULT NULL,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2020-10-26  9:40:03
