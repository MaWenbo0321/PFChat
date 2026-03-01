-- MySQL dump 10.13  Distrib 8.0.44, for Linux (x86_64)
--
-- Host: localhost    Database: im_system
-- ------------------------------------------------------
-- Server version	8.0.44

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `grammar_errors`
--

DROP TABLE IF EXISTS `grammar_errors`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `grammar_errors` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `message_id` bigint unsigned DEFAULT '0',
  `original_text` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `llm_suggestion` text COLLATE utf8mb4_unicode_ci,
  `llm_explanation` text COLLATE utf8mb4_unicode_ci,
  `error_type` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_grammar_errors_user_id` (`user_id`),
  KEY `idx_grammar_errors_error_type` (`error_type`)
) ENGINE=InnoDB AUTO_INCREMENT=19 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `grammar_errors`
--

LOCK TABLES `grammar_errors` WRITE;
/*!40000 ALTER TABLE `grammar_errors` DISABLE KEYS */;
INSERT INTO `grammar_errors` VALUES (7,4,7,'哇! 你会说日语!','哇！你还会说日语呀！','发送者是日本国籍，使用日语向中国接收者问候，接收者回应了日语。当前消息表达惊讶时语气略显突兀，缺乏对对方语言能力的尊重和礼貌缓冲，可能让对方感到被冒犯。应使用更委婉、带有赞赏语气的表达方式。“还会说”比“会说”更体现对方掌握日语的额外能力，语气更积极友善。','语言语用失误','2026-01-23 16:01:32.917'),(8,5,8,'Hey, bro what\'s up!','Hi Ma Wenbo, how are you doing?','The use of \'Hey, bro\' is too informal and may be considered overly familiar in Chinese culture, especially when addressing someone who may not be a close friend. In Chinese social norms, using a person\'s name directly without an appropriate title or level of formality can be acceptable in casual contexts, but \'bro\' introduces an unwarranted level of assumed intimacy. This constitutes a sociopragmatic failure because it violates cultural expectations regarding interpersonal distance and appropriateness of address.','社会语用失误','2026-01-23 20:29:39.999'),(9,4,0,'g','你好！','发送者是日本用户（国籍：Japan），但当前输入仅发送单个英文字母\'g\'，在日语或中日跨文化交际语境中无明确语义，无法构成有效言语行为；该表达缺乏基本的礼貌标记、可理解性和交际意图，属于言语行为形式严重缺失，违反基本会话合作原则（Grice\'s maxim of manner & quantity），构成语言语用失误。','语言语用失误','2026-02-20 14:50:13.571'),(10,4,0,'good morning!','Good afternoon!','对话发生在下午（前一条消息明确使用了\'Good afternoon!\'），而当前消息却使用\'good morning!\'，时间称谓与实际语境严重不符，导致言语行为失当（问候语未能准确反映交际时序），属于对言语行为形式（问候语的时间适配性）的误用，符合Thomas定义的语言语用失误。','语言语用失误','2026-02-20 14:50:28.283'),(11,4,0,'good morni!','Good morning!','当前输入 \'good morni!\' 是 \'Good morning!\' 的拼写错误，且末尾多了一个感叹号但缺少关键字母 \'g\'，导致语义不完整、形式不规范。在已知对话中双方正在进行跨文化礼貌问候（发送者为日本籍，接收者为中国籍），且前序消息已使用标准英语问候（\'Good afternoon!\'），此错误破坏了言语行为的得体性与可理解性，属于对英语礼貌问候形式的不当使用，符合Thomas定义的语言语用失误：即因语言形式（拼写、构形）错误导致言语行为失败，影响交际效果。','语言语用失误','2026-02-20 14:51:09.620'),(12,4,0,'good morning!','Good afternoon!','发送者（日本籍）在对话中先前已使用\'Good afternoon!\'问候，接收者（中国籍）以日语\'こんにちは！\'回应，表明当前时间仍为下午。随后发送者突然切换为\'good morning!\'，与实际时间及上下文不一致，造成语用不协调；该失误源于对言语行为适切性（即问候语需与时境匹配）的忽视，属于言语行为形式选择不当。','语言语用失误','2026-02-20 14:51:13.680'),(13,1,0,'How\'s','How\'s it going?','当前输入 \'How\'s\' 是一个不完整的句子，属于截断式表达，在非亲密熟人之间的初始对话中显得突兀、不礼貌且缺乏基本语用完整性；结合对话历史（双方刚互致问候并发现语言能力惊喜），此时突然抛出不完整问句，既未延续日语问候的礼貌语境，也未提供足够信息供对方回应，构成言语行为实施失败——即使用了语法上可能存在的缩略形式，但该形式在该语境中无法恰当执行‘问候/开启话题’这一言语行为，属于Thomas定义的语言语用失误。','语言语用失误','2026-02-20 14:53:40.859'),(14,1,0,'How\'s ev','How\'s everything going?','当前输入 \'How\'s ev\' 是不完整的缩写，属于非正式口语中常见的打字省略，但在初次跨文化寒暄语境中（尤其对方刚惊喜于发送者会日语），该截断形式显得随意、敷衍甚至不礼貌，削弱了问候的诚意和完整性；它不符合英语日常社交问候的基本语用规范，属于言语行为形式选择不当。','语言语用失误','2026-02-20 14:53:44.872'),(15,1,0,'How\'s ever','How\'s everything going?','‘How\'s ever’ 是不完整的、非标准的英语表达，属于语法和语用层面的误用。它试图缩略 ‘How\'s everything’ 或 ‘How\'s every thing’，但实际不存在这一固定表达；母语者不会使用该形式。在跨文化交际中，这种不规范表达可能造成理解困难或显得语言能力不足，影响交际效果。该失误源于对英语惯用语形式（如 ‘How\'s everything going?’）的错误简化，属于言语行为实现手段不当，符合Thomas定义的语言语用失误。','语言语用失误','2026-02-20 14:53:50.654'),(16,1,0,'How\'s everything g','How\'s everything going?','当前输入 \'How\'s everything g\' 是一个不完整的句子，\'g\' 显然是 \'going\' 的未完成输入，属于打字中断或输入错误。在跨文化交际中，尤其面对日本接收者（重视语言准确性和礼貌性），这种不完整、非规范的表达可能被解读为随意、不认真或缺乏基本语言素养，削弱话语的礼貌性与可理解性，构成言语行为执行不当——即使用了有缺陷的语言形式来实施问候/寒暄这一言语行为。','语言语用失误','2026-02-20 14:53:55.543'),(17,4,0,'goo','Good afternoon!','发送者是日本籍（Japan），但当前输入\'goo\'是不完整的英文问候语，明显为\'Good afternoon!\'的拼写错误或截断。该形式不符合基本语言规范，导致言语行为（问候）无法被正确识别和理解，属于言语行为表达形式不当，构成语言语用失误。','语言语用失误','2026-02-20 15:39:52.460'),(18,4,9,'good morning','Good afternoon','对话历史中发送者先说\'Hi! Good afternoon!\'，接收者用日语回应\'こんにちは！\'（适用于白天的问候），当前消息\'good morning\'与实际时间（下午）及上下文问候时段矛盾，造成语用不协调；该失误源于对言语行为适切性（即问候语需匹配真实时间/语境）的忽视，属于言语行为形式选择不当，符合Thomas定义的语言语用失误。','语言语用失误','2026-02-20 15:39:57.894');
/*!40000 ALTER TABLE `grammar_errors` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `messages`
--

DROP TABLE IF EXISTS `messages`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `messages` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `sender_id` bigint unsigned NOT NULL,
  `receiver_id` bigint unsigned NOT NULL,
  `content` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `sent_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_messages_sender_id` (`sender_id`),
  KEY `idx_messages_receiver_id` (`receiver_id`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `messages`
--

LOCK TABLES `messages` WRITE;
/*!40000 ALTER TABLE `messages` DISABLE KEYS */;
INSERT INTO `messages` VALUES (1,2,1,'Hi Wenbo','2026-01-22 09:48:45.693',NULL),(2,2,1,'The Lunar New Year is approaching. Do you have any plan?','2026-01-22 09:53:12.408',NULL),(3,2,1,'Hi Wenbo. How are you?','2026-01-22 09:58:58.110',NULL),(4,2,1,'Would you please introduce your research to me?','2026-01-22 10:00:02.764',NULL),(5,4,1,'Hi! Good afternoon!','2026-01-23 15:58:47.040',NULL),(6,1,4,'こんにちは！','2026-01-23 16:00:16.980',NULL),(7,4,1,'哇! 你会说日语!','2026-01-23 16:01:47.122',NULL),(8,5,1,'Hey, bro what\'s up!','2026-01-23 20:29:42.911',NULL),(9,4,1,'good morning','2026-02-20 15:40:05.108',NULL);
/*!40000 ALTER TABLE `messages` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `password` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `country` varchar(191) COLLATE utf8mb4_unicode_ci NOT NULL,
  `role` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT 'user',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`),
  KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `users`
--

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
INSERT INTO `users` VALUES (1,'Ma Wenbo','$2a$10$zGoIVEOOlZPfzyWlzlgtJuoVbuxm/AwRWW7jFDOW6IQxh6XZI4/k.','CN','user','2025-12-29 19:06:14.625','2025-12-29 19:06:14.625',NULL),(2,'zhuying','$2a$10$gDIuiLqGRgTSf.0s2P06BuLDO41HwK89.VxWlyK7L.zoof4/zvhri','CN','user','2026-01-22 09:48:18.546','2026-01-22 09:48:18.546',NULL),(3,'admin','$2a$10$TWQCwP1c/N7LOOld5fxc5OJPTmSz1KVaJNXOWk17Pk4BtYPK75p5W','CN','admin','2026-01-23 15:44:17.256','2026-01-23 15:44:17.256',NULL),(4,'test','$2a$10$Ggh38ii7hTcGYEZ6wo2o5OL42bryJxF4nC5EupFOA/On3LipAmynC','JP','user','2026-01-23 15:57:43.631','2026-01-23 15:57:43.631',NULL),(5,'test_2','$2a$10$FfI9sOxc.D1jlr6VuikjvOS7knX/jiLG58NAJN1yog847p0cAloVu','US','user','2026-01-23 20:01:28.482','2026-01-23 20:01:28.482',NULL);
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2026-02-26 15:24:16
